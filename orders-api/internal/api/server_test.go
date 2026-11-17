package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dhaba/orders-api/internal/order"
)

type fakeStore struct {
	orders  []order.Order
	failOn  string
	created int
}

func (f *fakeStore) Create(_ context.Context, o *order.Order) error {
	if f.failOn == "create" {
		return errors.New("db down")
	}
	f.created++
	o.ID = int64(len(f.orders) + 1)
	o.Status = order.StatusNew
	o.CreatedAt = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	f.orders = append(f.orders, *o)
	return nil
}

func (f *fakeStore) Get(_ context.Context, id int64) (*order.Order, error) {
	if f.failOn == "get" {
		return nil, errors.New("db down")
	}
	for _, o := range f.orders {
		if o.ID == id {
			return &o, nil
		}
	}
	return nil, order.ErrNotFound
}

func (f *fakeStore) List(_ context.Context, limit int) ([]order.Order, error) {
	if f.failOn == "list" {
		return nil, errors.New("db down")
	}
	out := []order.Order{}
	for i := len(f.orders) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, f.orders[i])
	}
	return out, nil
}

type fakeQueue struct {
	ids []int64
	err error
}

func (q *fakeQueue) Enqueue(_ context.Context, id int64) error {
	if q.err != nil {
		return q.err
	}
	q.ids = append(q.ids, id)
	return nil
}

type fakeStats struct {
	calls [][]order.Item
	at    []time.Time
	err   error
}

func (s *fakeStats) RecordOrder(_ context.Context, items []order.Item, at time.Time) error {
	s.calls = append(s.calls, items)
	s.at = append(s.at, at)
	return s.err
}

type harness struct {
	srv   *Server
	store *fakeStore
	queue *fakeQueue
	stats *fakeStats
}

func newHarness() *harness {
	h := &harness{store: &fakeStore{}, queue: &fakeQueue{}, stats: &fakeStats{}}
	h.srv = &Server{
		Store:   h.store,
		Queue:   h.queue,
		Stats:   h.stats,
		Version: "1.2.3",
		Pod:     "orders-api-abc",
		Now:     func() time.Time { return time.Date(2026, 11, 17, 12, 0, 30, 0, time.UTC) },
	}
	return h
}

func (h *harness) do(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, req)
	return rec
}

const validBody = `{"customer":"  Ali ","items":[{"name":"Chicken Karahi","qty":2,"price":12.5},{"name":"Naan","qty":4}]}`

func TestHealth(t *testing.T) {
	rec := newHarness().do("GET", "/health", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestInfoShowsVersionAndPod(t *testing.T) {
	rec := newHarness().do("GET", "/info", "")
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || got["version"] != "1.2.3" || got["pod"] != "orders-api-abc" || got["service"] != "orders-api" {
		t.Fatalf("got %d %v", rec.Code, got)
	}
}

func TestReady(t *testing.T) {
	h := newHarness()
	h.srv.Checks = map[string]func(context.Context) error{
		"postgres": func(context.Context) error { return nil },
		"redis":    func(context.Context) error { return nil },
	}
	if rec := h.do("GET", "/ready", ""); rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	h.srv.Checks["redis"] = func(context.Context) error { return errors.New("connection refused") }
	rec := h.do("GET", "/ready", "")
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("expected 503 naming the failure, got %d %s", rec.Code, rec.Body)
	}
}

func TestCreateOrderSuccess(t *testing.T) {
	h := newHarness()
	rec := h.do("POST", "/orders", validBody)
	if rec.Code != 201 {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	var got order.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 || got.Status != "new" || got.Customer != "Ali" || len(got.Items) != 2 {
		t.Fatalf("unexpected order %+v", got)
	}
	if len(h.queue.ids) != 1 || h.queue.ids[0] != 1 {
		t.Fatalf("expected order 1 queued, got %v", h.queue.ids)
	}
	if len(h.stats.calls) != 1 || len(h.stats.calls[0]) != 2 {
		t.Fatalf("expected stats recorded once with 2 items, got %v", h.stats.calls)
	}
	if !h.stats.at[0].Equal(time.Date(2026, 11, 17, 12, 0, 30, 0, time.UTC)) {
		t.Fatalf("stats recorded with wrong time %v", h.stats.at[0])
	}
}

func TestCreateOrderValidation(t *testing.T) {
	cases := map[string]string{
		"bad json":          `{not json`,
		"empty body":        ``,
		"no customer":       `{"items":[{"name":"Dal","qty":1}]}`,
		"blank customer":    `{"customer":"   ","items":[{"name":"Dal","qty":1}]}`,
		"no items":          `{"customer":"Ali","items":[]}`,
		"blank item name":   `{"customer":"Ali","items":[{"name":" ","qty":1}]}`,
		"zero qty":          `{"customer":"Ali","items":[{"name":"Dal","qty":0}]}`,
		"huge qty":          `{"customer":"Ali","items":[{"name":"Dal","qty":100}]}`,
		"negative price":    `{"customer":"Ali","items":[{"name":"Dal","qty":1,"price":-1}]}`,
		"qty wrong type":    `{"customer":"Ali","items":[{"name":"Dal","qty":"two"}]}`,
		"too many items":    `{"customer":"Ali","items":` + manyItems(51) + `}`,
		"customer too long": `{"customer":"` + strings.Repeat("a", 101) + `","items":[{"name":"Dal","qty":1}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness()
			rec := h.do("POST", "/orders", body)
			if rec.Code != 400 {
				t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body)
			}
			if h.store.created != 0 || len(h.queue.ids) != 0 {
				t.Fatal("invalid order must not be stored or queued")
			}
		})
	}
}

func manyItems(n int) string {
	items := make([]order.Item, n)
	for i := range items {
		items[i] = order.Item{Name: "Dal", Qty: 1}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func TestCreateOrderStoreFailure(t *testing.T) {
	h := newHarness()
	h.store.failOn = "create"
	rec := h.do("POST", "/orders", validBody)
	if rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if len(h.queue.ids) != 0 {
		t.Fatal("nothing should be queued when saving fails")
	}
}

func TestCreateOrderQueueFailure(t *testing.T) {
	h := newHarness()
	h.queue.err = errors.New("redis down")
	rec := h.do("POST", "/orders", validBody)
	if rec.Code != 503 {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestCreateOrderStatsFailureIsNotFatal(t *testing.T) {
	h := newHarness()
	h.stats.err = errors.New("redis down")
	rec := h.do("POST", "/orders", validBody)
	if rec.Code != 201 {
		t.Fatalf("expected 201 even if stats fail, got %d", rec.Code)
	}
}

func TestGetOrder(t *testing.T) {
	h := newHarness()
	h.do("POST", "/orders", validBody)

	if rec := h.do("GET", "/orders/1", ""); rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec := h.do("GET", "/orders/99", ""); rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	for _, id := range []string{"abc", "0", "-3"} {
		if rec := h.do("GET", "/orders/"+id, ""); rec.Code != 400 {
			t.Fatalf("id %q: expected 400, got %d", id, rec.Code)
		}
	}
	h.store.failOn = "get"
	if rec := h.do("GET", "/orders/1", ""); rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListOrders(t *testing.T) {
	h := newHarness()
	for i := 0; i < 3; i++ {
		h.do("POST", "/orders", validBody)
	}

	rec := h.do("GET", "/orders", "")
	var all []order.Order
	_ = json.Unmarshal(rec.Body.Bytes(), &all)
	if rec.Code != 200 || len(all) != 3 || all[0].ID != 3 {
		t.Fatalf("expected 3 orders newest first, got %d %+v", rec.Code, all)
	}

	rec = h.do("GET", "/orders?limit=2", "")
	var two []order.Order
	_ = json.Unmarshal(rec.Body.Bytes(), &two)
	if len(two) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(two))
	}

	for _, q := range []string{"limit=0", "limit=x", "limit=-1"} {
		if rec := h.do("GET", "/orders?"+q, ""); rec.Code != 400 {
			t.Fatalf("%s: expected 400, got %d", q, rec.Code)
		}
	}
}

func TestListOrdersEmptyIsJSONArray(t *testing.T) {
	rec := newHarness().do("GET", "/orders", "")
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("expected [], got %s", rec.Body)
	}
}

func TestCORS(t *testing.T) {
	h := newHarness()
	h.srv.CORSOrigin = "*"

	req := httptest.NewRequest("OPTIONS", "/orders", bytes.NewReader(nil))
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("preflight failed: %d %v", rec.Code, rec.Header())
	}

	rec = h.do("GET", "/health", "")
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("normal responses should carry the CORS header")
	}

	h.srv.CORSOrigin = ""
	rec = h.do("GET", "/health", "")
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CORS header should be absent when disabled")
	}
}

func TestUnknownRouteAndWrongMethod(t *testing.T) {
	h := newHarness()
	if rec := h.do("GET", "/nope", ""); rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if rec := h.do("DELETE", "/orders/1", ""); rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
