// Package order holds the order model shared by the HTTP layer and the store.
package order

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	StatusNew     = "new"
	StatusCooking = "cooking"
	StatusReady   = "ready"
)

const (
	maxItems    = 50
	maxQty      = 99
	maxNameLen  = 100
	maxCustomer = 100
)

// ErrNotFound is returned by stores when an order does not exist.
var ErrNotFound = errors.New("order not found")

type Item struct {
	Name  string  `json:"name"`
	Qty   int     `json:"qty"`
	Price float64 `json:"price,omitempty"`
}

type Order struct {
	ID        int64     `json:"id"`
	Customer  string    `json:"customer"`
	Items     []Item    `json:"items"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Validate trims whitespace in place and checks the order is acceptable.
func (o *Order) Validate() error {
	o.Customer = strings.TrimSpace(o.Customer)
	if o.Customer == "" {
		return errors.New("customer is required")
	}
	if len(o.Customer) > maxCustomer {
		return fmt.Errorf("customer must be at most %d characters", maxCustomer)
	}
	if len(o.Items) == 0 {
		return errors.New("at least one item is required")
	}
	if len(o.Items) > maxItems {
		return fmt.Errorf("at most %d items per order", maxItems)
	}
	for i := range o.Items {
		it := &o.Items[i]
		it.Name = strings.TrimSpace(it.Name)
		if it.Name == "" {
			return fmt.Errorf("items[%d].name is required", i)
		}
		if len(it.Name) > maxNameLen {
			return fmt.Errorf("items[%d].name must be at most %d characters", i, maxNameLen)
		}
		if it.Qty < 1 || it.Qty > maxQty {
			return fmt.Errorf("items[%d].qty must be between 1 and %d", i, maxQty)
		}
		if it.Price < 0 {
			return fmt.Errorf("items[%d].price must not be negative", i)
		}
	}
	return nil
}
