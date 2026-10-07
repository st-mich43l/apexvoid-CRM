package health

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Status struct {
	Postgres string `json:"postgres"`
}

type Checker struct{ postgres *pgxpool.Pool }

func NewChecker(postgres *pgxpool.Pool) *Checker {
	return &Checker{postgres: postgres}
}

func (c *Checker) Check(ctx context.Context) (Status, error) {
	status := Status{Postgres: "ok"}
	if err := c.postgres.Ping(ctx); err != nil {
		status.Postgres = "unavailable"
		return status, fmt.Errorf("postgres: %w", err)
	}
	return status, nil
}
