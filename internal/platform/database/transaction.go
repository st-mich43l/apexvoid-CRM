package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager struct{ pool *pgxpool.Pool }

func NewTxManager(pool *pgxpool.Pool) *TxManager { return &TxManager{pool: pool} }

// WithTransaction makes the application use case the transaction boundary.
// Repositories can retrieve the active pgx transaction explicitly through
// TransactionFromContext without starting nested transactions themselves.
func (m *TxManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	if _, active := TransactionFromContext(ctx); active {
		return fn(ctx)
	}
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	txCtx := context.WithValue(ctx, transactionKey{}, tx)
	hooks := &commitHooks{}
	txCtx = context.WithValue(txCtx, afterCommitKey{}, hooks)
	defer func() { _ = tx.Rollback(txCtx) }()
	if err := fn(txCtx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	// Event callbacks never see the finished transaction, nor a cancelled request.
	postCommitCtx := context.WithValue(context.WithoutCancel(ctx), transactionKey{}, nil)
	for _, callback := range hooks.drain() {
		callback(postCommitCtx)
	}
	return nil
}

func TransactionFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(transactionKey{}).(pgx.Tx)
	return tx, ok
}

type transactionKey struct{}

type afterCommitKey struct{}

type commitHooks struct {
	mu        sync.Mutex
	callbacks []func(context.Context)
}

func (h *commitHooks) add(callback func(context.Context)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callbacks = append(h.callbacks, callback)
}

func (h *commitHooks) drain() []func(context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	callbacks := h.callbacks
	h.callbacks = nil
	return callbacks
}

// AfterCommit defers an effect until the outermost transaction commits.
// A failed or rolled-back transaction discards its registered callbacks.
// Without a managed transaction the caller must already have committed its write.
func AfterCommit(ctx context.Context, callback func(context.Context)) {
	if hooks, ok := ctx.Value(afterCommitKey{}).(*commitHooks); ok {
		hooks.add(callback)
		return
	}
	callback(ctx)
}
