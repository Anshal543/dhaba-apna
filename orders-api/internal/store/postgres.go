// Package store is the Postgres implementation of the orders store.
// The SQL driver (github.com/lib/pq) is registered by main, not here.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"dhaba/orders-api/internal/order"
)

type Postgres struct {
	DB *sql.DB
}

func NewPostgres(db *sql.DB) *Postgres { return &Postgres{DB: db} }

func (p *Postgres) Ping(ctx context.Context) error { return p.DB.PingContext(ctx) }

// Create inserts the order with status "new" and fills ID, Status and CreatedAt.
func (p *Postgres) Create(ctx context.Context, o *order.Order) error {
	items, err := json.Marshal(o.Items)
	if err != nil {
		return err
	}
	o.Status = order.StatusNew
	return p.DB.QueryRowContext(ctx,
		`INSERT INTO orders (customer, items, status) VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		o.Customer, items, o.Status,
	).Scan(&o.ID, &o.CreatedAt)
}

func (p *Postgres) Get(ctx context.Context, id int64) (*order.Order, error) {
	row := p.DB.QueryRowContext(ctx,
		`SELECT id, customer, items, status, created_at FROM orders WHERE id = $1`, id)
	o, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, order.ErrNotFound
	}
	return o, err
}

func (p *Postgres) List(ctx context.Context, limit int) ([]order.Order, error) {
	rows, err := p.DB.QueryContext(ctx,
		`SELECT id, customer, items, status, created_at FROM orders
		 ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []order.Order{}
	for rows.Next() {
		o, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scan(s scanner) (*order.Order, error) {
	var (
		o     order.Order
		items []byte
	)
	if err := s.Scan(&o.ID, &o.Customer, &items, &o.Status, &o.CreatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(items, &o.Items); err != nil {
		return nil, err
	}
	return &o, nil
}
