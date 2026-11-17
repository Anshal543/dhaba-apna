// Package api is the HTTP layer of orders-api. It depends only on small
// interfaces, so it can be tested without Postgres or Redis.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"dhaba/orders-api/internal/order"
)

type Store interface {
	Create(ctx context.Context, o *order.Order) error
	Get(ctx context.Context, id int64) (*order.Order, error)
	List(ctx context.Context, limit int) ([]order.Order, error)
}

type Queue interface {
	Enqueue(ctx context.Context, orderID int64) error
}

type Stats interface {
	RecordOrder(ctx context.Context, items []order.Item, at time.Time) error
}

type Server struct {
	Store   Store
	Queue   Queue
	Stats   Stats
	Version string
	Pod     string
	// Origin for the Access-Control-Allow-Origin header. Empty disables CORS.
	CORSOrigin string
	// Dependency checks used by /ready.
	Checks map[string]func(ctx context.Context) error
	Now    func() time.Time
}

const (
	defaultListLimit = 50
	maxListLimit     = 200
	maxBodyBytes     = 1 << 20
)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("GET /info", s.info)
	mux.HandleFunc("POST /orders", s.createOrder)
	mux.HandleFunc("GET /orders", s.listOrders)
	mux.HandleFunc("GET /orders/{id}", s.getOrder)
	return s.cors(mux)
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	failed := map[string]string{}
	for name, check := range s.Checks {
		if err := check(ctx); err != nil {
			failed[name] = err.Error()
		}
	}
	if len(failed) > 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "failed": failed})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "orders-api",
		"version": s.Version,
		"pod":     s.Pod,
	})
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var o order.Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := o.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.Store.Create(r.Context(), &o); err != nil {
		log.Printf("create order: %v", err)
		writeError(w, http.StatusInternalServerError, "could not save order")
		return
	}
	if err := s.Queue.Enqueue(r.Context(), o.ID); err != nil {
		log.Printf("enqueue order %d: %v", o.ID, err)
		writeError(w, http.StatusServiceUnavailable, "order saved but could not be queued")
		return
	}
	// Stats are best effort: never fail an order because the counters are down.
	if err := s.Stats.RecordOrder(r.Context(), o.Items, s.now()); err != nil {
		log.Printf("record stats for order %d: %v", o.ID, err)
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	o, err := s.Store.Get(r.Context(), id)
	if errors.Is(err, order.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		log.Printf("get order %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "could not load order")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	limit := defaultListLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = min(n, maxListLimit)
	}
	orders, err := s.Store.List(r.Context(), limit)
	if err != nil {
		log.Printf("list orders: %v", err)
		writeError(w, http.StatusInternalServerError, "could not list orders")
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) cors(next http.Handler) http.Handler {
	if s.CORSOrigin == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", s.CORSOrigin)
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
