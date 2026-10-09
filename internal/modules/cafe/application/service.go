package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/cafe/domain"
	erpapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Service struct {
	pool     *pgxpool.Pool
	tx       *database.TxManager
	products erpapi.ProductReader
}

func New(pool *pgxpool.Pool, tx *database.TxManager, products erpapi.ProductReader) *Service {
	return &Service{pool: pool, tx: tx, products: products}
}
func (s *Service) q(ctx context.Context) queryer {
	if tx, ok := database.TransactionFromContext(ctx); ok {
		return tx
	}
	return s.pool
}
func mapped(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01", "23503":
			return domain.ErrConflict
		case "23514", "22003":
			return domain.ErrInvalid
		}
	}
	return err
}
func (s *Service) CreateBooth(ctx context.Context, w, u uuid.UUID, name string) (domain.Booth, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 100 {
		return domain.Booth{}, domain.ErrInvalid
	}
	var out domain.Booth
	err := s.q(ctx).QueryRow(ctx, `INSERT INTO cafe_booths(id,workspace_id,name,created_by) VALUES($1,$2,$3,$4) RETURNING id,workspace_id,name,active,created_at`, uuid.New(), w, name, u).Scan(&out.ID, &out.WorkspaceID, &out.Name, &out.Active, &out.CreatedAt)
	return out, mapped(err)
}
func (s *Service) ListBooths(ctx context.Context, w uuid.UUID) ([]domain.Booth, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT id,workspace_id,name,active,created_at FROM cafe_booths WHERE workspace_id=$1 ORDER BY name`, w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Booth{}
	for rows.Next() {
		var booth domain.Booth
		if err := rows.Scan(&booth.ID, &booth.WorkspaceID, &booth.Name, &booth.Active, &booth.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, booth)
	}
	return out, rows.Err()
}
func scanBooking(row pgx.Row) (domain.Booking, error) {
	var b domain.Booking
	err := row.Scan(&b.ID, &b.WorkspaceID, &b.BoothID, &b.BoothName, &b.PackageProductID, &b.GuestName, &b.StartsAt, &b.EndsAt, &b.Status, &b.Price, &b.Currency, &b.CreatedAt)
	return b, mapped(err)
}

const bookingColumns = "b.id,b.workspace_id,b.booth_id,t.name,b.package_product_id,b.guest_name,b.starts_at,b.ends_at,b.status,b.price::text,b.currency,b.created_at"

func (s *Service) Booking(ctx context.Context, w, id uuid.UUID) (domain.Booking, error) {
	return scanBooking(s.q(ctx).QueryRow(ctx, `SELECT `+bookingColumns+` FROM cafe_bookings b JOIN cafe_booths t ON t.id=b.booth_id AND t.workspace_id=b.workspace_id WHERE b.workspace_id=$1 AND b.id=$2`, w, id))
}
func (s *Service) ListBookings(ctx context.Context, w uuid.UUID) ([]domain.Booking, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT `+bookingColumns+` FROM cafe_bookings b JOIN cafe_booths t ON t.id=b.booth_id AND t.workspace_id=b.workspace_id WHERE b.workspace_id=$1 ORDER BY b.starts_at DESC LIMIT 100`, w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Booking{}
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(&b.ID, &b.WorkspaceID, &b.BoothID, &b.BoothName, &b.PackageProductID, &b.GuestName, &b.StartsAt, &b.EndsAt, &b.Status, &b.Price, &b.Currency, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (s *Service) Reserve(ctx context.Context, w, u uuid.UUID, input domain.BookingInput) (domain.Booking, error) {
	input, err := input.Validate(time.Now().UTC())
	if err != nil {
		return domain.Booking{}, err
	}
	product, err := s.products.Get(ctx, w, input.PackageProductID)
	if err != nil {
		return domain.Booking{}, domain.ErrInvalid
	}
	if product.Kind != "service" || product.Status != "active" {
		return domain.Booking{}, domain.ErrInvalid
	}
	id := uuid.New()
	// Atomic INSERT/SELECT prevents reserving inactive or cross-workspace booths.
	tag, err := s.q(ctx).Exec(ctx, `INSERT INTO cafe_bookings(id,workspace_id,booth_id,package_product_id,guest_name,starts_at,ends_at,price,currency,created_by)
		SELECT $1,$2,booth.id,$3,$4,$5,$6,$7::numeric,$8,$9
		FROM cafe_booths booth WHERE booth.id=$10 AND booth.workspace_id=$2 AND booth.active`,
		id, w, input.PackageProductID, input.GuestName, input.StartsAt, input.EndsAt, product.UnitPrice, product.Currency, u, input.BoothID)
	if err != nil {
		return domain.Booking{}, mapped(err)
	}
	if tag.RowsAffected() != 1 {
		return domain.Booking{}, domain.ErrInvalid
	}
	return s.Booking(ctx, w, id)
}
func (s *Service) TransitionBooking(ctx context.Context, w, id uuid.UUID, action string) (domain.Booking, error) {
	var from, to string
	switch action {
	case "check-in":
		from, to = "reserved", "checked_in"
	case "complete":
		from, to = "checked_in", "completed"
	case "cancel":
		from, to = "reserved", "cancelled"
	default:
		return domain.Booking{}, domain.ErrInvalid
	}
	tag, err := s.q(ctx).Exec(ctx, `UPDATE cafe_bookings SET status=$3,updated_at=NOW() WHERE workspace_id=$1 AND id=$2 AND status=$4`, w, id, to, from)
	if err != nil {
		return domain.Booking{}, mapped(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Booking{}, domain.ErrConflict
	}
	return s.Booking(ctx, w, id)
}
func scanOrder(row pgx.Row) (domain.Order, error) {
	var o domain.Order
	err := row.Scan(&o.ID, &o.WorkspaceID, &o.Status, &o.Total, &o.Currency, &o.Note, &o.LineCount, &o.CreatedAt)
	return o, mapped(err)
}

const orderColumns = "o.id,o.workspace_id,o.status,o.total::text,o.currency,o.note,(SELECT count(*)::integer FROM cafe_order_lines l WHERE l.workspace_id=o.workspace_id AND l.order_id=o.id),o.created_at"

func (s *Service) Order(ctx context.Context, w, id uuid.UUID) (domain.Order, error) {
	return scanOrder(s.q(ctx).QueryRow(ctx, `SELECT `+orderColumns+` FROM cafe_orders o WHERE o.workspace_id=$1 AND o.id=$2`, w, id))
}
func (s *Service) ListOrders(ctx context.Context, w uuid.UUID) ([]domain.Order, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT `+orderColumns+` FROM cafe_orders o WHERE o.workspace_id=$1 ORDER BY o.created_at DESC LIMIT 100`, w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Order{}
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.WorkspaceID, &o.Status, &o.Total, &o.Currency, &o.Note, &o.LineCount, &o.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, rows.Err()
}
func (s *Service) CreateOrder(ctx context.Context, w, u uuid.UUID, input domain.OrderInput) (domain.Order, error) {
	input, err := input.Validate()
	if err != nil {
		return domain.Order{}, err
	}
	type detail struct {
		id          uuid.UUID
		quantity    int
		name, price string
	}
	details := make([]detail, 0, len(input.Lines))
	currency := ""
	for _, line := range input.Lines {
		product, err := s.products.Get(ctx, w, line.ProductID)
		if err != nil || product.Status != "active" || product.Kind != "good" {
			return domain.Order{}, domain.ErrInvalid
		}
		if currency != "" && currency != product.Currency {
			return domain.Order{}, domain.ErrInvalid
		}
		currency = product.Currency
		details = append(details, detail{id: product.ID, quantity: line.Quantity, name: product.Name, price: product.UnitPrice})
	}
	id := uuid.New()
	var result domain.Order
	err = s.tx.WithTransaction(ctx, func(txCtx context.Context) error {
		q := s.q(txCtx)
		if _, err := q.Exec(txCtx, `INSERT INTO cafe_orders(id,workspace_id,currency,note,created_by) VALUES($1,$2,$3,$4,$5)`, id, w, currency, input.Note, u); err != nil {
			return mapped(err)
		}
		for _, line := range details {
			_, err := q.Exec(txCtx, `INSERT INTO cafe_order_lines(id,workspace_id,order_id,product_id,product_name,quantity,unit_price) VALUES($1,$2,$3,$4,$5,$6,$7::numeric)`,
				uuid.New(), w, id, line.id, line.name, line.quantity, line.price)
			if err != nil {
				return mapped(err)
			}
		}
		_, err := q.Exec(txCtx, `UPDATE cafe_orders SET total=(SELECT coalesce(sum(line_total),0) FROM cafe_order_lines WHERE workspace_id=$1 AND order_id=$2) WHERE workspace_id=$1 AND id=$2`, w, id)
		if err != nil {
			return mapped(err)
		}
		result, err = s.Order(txCtx, w, id)
		return err
	})
	return result, err
}
func (s *Service) TransitionOrder(ctx context.Context, w, id uuid.UUID, action string) (domain.Order, error) {
	status := ""
	switch action {
	case "complete":
		status = "completed"
	case "cancel":
		status = "cancelled"
	default:
		return domain.Order{}, domain.ErrInvalid
	}
	tag, err := s.q(ctx).Exec(ctx, `UPDATE cafe_orders SET status=$3,updated_at=NOW() WHERE workspace_id=$1 AND id=$2 AND status='open'`, w, id, status)
	if err != nil {
		return domain.Order{}, mapped(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Order{}, domain.ErrConflict
	}
	return s.Order(ctx, w, id)
}
