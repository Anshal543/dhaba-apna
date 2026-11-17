package redisq

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"dhaba/orders-api/internal/order"
)

// Publisher puts jobs on the kitchen queue and records live stats.
type Publisher struct {
	Client *Client
}

func NewPublisher(c *Client) *Publisher { return &Publisher{Client: c} }

// Enqueue pushes a job for the kitchen worker. Job format: {"order_id": 42}.
func (p *Publisher) Enqueue(ctx context.Context, orderID int64) error {
	job := fmt.Sprintf(`{"order_id":%d}`, orderID)
	_, err := p.Client.Do(ctx, "LPUSH", QueueKey, job)
	return err
}

// RecordOrder bumps the per-minute counter and the per-dish totals.
func (p *Publisher) RecordOrder(ctx context.Context, items []order.Item, at time.Time) error {
	minuteKey := fmt.Sprintf(MinuteKeyFmt, at.Unix()/60)
	cmds := [][]string{
		{"INCR", minuteKey},
		{"EXPIRE", minuteKey, strconv.Itoa(minuteKeyTTL)},
	}
	for _, it := range items {
		cmds = append(cmds, []string{"ZINCRBY", DishesKey, strconv.Itoa(it.Qty), it.Name})
	}
	_, err := p.Client.Pipeline(ctx, cmds...)
	return err
}

func (p *Publisher) Ping(ctx context.Context) error { return p.Client.Ping(ctx) }
