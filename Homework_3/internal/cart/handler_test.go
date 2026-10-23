package cart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"homework2/internal/model"
	"homework2/pkg/logger"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCatalog struct {
	mu       sync.Mutex
	products map[int64]model.Product
	err      error
	calls    int
}

func (f *fakeCatalog) GetProduct(ctx context.Context, sku int64) (model.Product, error) {
	if err := ctx.Err(); err != nil {
		return model.Product{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return model.Product{}, f.err
	}
	p, ok := f.products[sku]
	if !ok {
		return model.Product{}, status.Error(codes.NotFound, "product not found")
	}
	return p, nil
}

type gateCatalog struct {
	mu       sync.Mutex
	products map[int64]model.Product
	hold     bool
	need     int
	n        int
	arrived  chan struct{}
	release  chan struct{}
}

func (g *gateCatalog) GetProduct(ctx context.Context, sku int64) (model.Product, error) {
	g.mu.Lock()
	hold := g.hold
	if hold {
		g.n++
		if g.n == g.need {
			close(g.arrived)
		}
	}
	p, ok := g.products[sku]
	g.mu.Unlock()

	if hold {
		select {
		case <-g.release:
		case <-ctx.Done():
			return model.Product{}, ctx.Err()
		}
	}
	if !ok {
		return model.Product{}, status.Error(codes.NotFound, "product not found")
	}
	return p, nil
}

func newHandler(products map[int64]model.Product) *Handler {
	copied := make(map[int64]model.Product, len(products))
	for k, v := range products {
		copied[k] = v
	}
	return NewHandler(NewStore(), &fakeCatalog{products: copied}, logger.New())
}

func TestAddItem(t *testing.T) {
	t.Parallel()

	catalog := map[int64]model.Product{
		2008: {SKU: 2008, Name: "Book", Price: 16},
	}
	tests := []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{name: "ok", path: "/user/1007/cart/2008", body: `{"count":2}`, status: http.StatusOK},
		{name: "zero user", path: "/user/0/cart/2008", body: `{"count":2}`, status: http.StatusBadRequest},
		{name: "negative user", path: "/user/-1/cart/2008", body: `{"count":2}`, status: http.StatusBadRequest},
		{name: "zero sku", path: "/user/1007/cart/0", body: `{"count":2}`, status: http.StatusBadRequest},
		{name: "negative sku", path: "/user/1007/cart/-5", body: `{"count":2}`, status: http.StatusBadRequest},
		{name: "zero count", path: "/user/1007/cart/2008", body: `{"count":0}`, status: http.StatusBadRequest},
		{name: "negative count", path: "/user/1007/cart/2008", body: `{"count":-1}`, status: http.StatusBadRequest},
		{name: "broken json", path: "/user/1007/cart/2008", body: `{`, status: http.StatusBadRequest},
		{name: "missing sku", path: "/user/1007/cart/9", body: `{"count":1}`, status: http.StatusPreconditionFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			newHandler(catalog).Routes().ServeHTTP(rr, req)
			require.Equal(t, tt.status, rr.Code)
		})
	}
}

func TestAddItemCatalogDown(t *testing.T) {
	t.Parallel()

	h := NewHandler(NewStore(), &fakeCatalog{err: status.Error(codes.Unavailable, "down")}, logger.New())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/user/1/cart/5", strings.NewReader(`{"count":1}`))
	h.Routes().ServeHTTP(rr, req)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestGetCartContract(t *testing.T) {
	t.Parallel()

	fake := &fakeCatalog{products: map[int64]model.Product{
		2: {SKU: 2, Name: "A", Price: 10},
		5: {SKU: 5, Name: "B", Price: 10},
	}}
	h := NewHandler(NewStore(), fake, logger.New())
	post := func(sku string, count int) {
		rr := httptest.NewRecorder()
		body := fmt.Sprintf(`{"count":%d}`, count)
		req := httptest.NewRequest(http.MethodPost, "/user/7/cart/"+sku, strings.NewReader(body))
		h.Routes().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}
	post("5", 2)
	post("2", 1)
	post("5", 1)

	fake.mu.Lock()
	item := fake.products[5]
	item.Price = 9
	fake.products[5] = item
	fake.mu.Unlock()

	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/user/7/cart", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	dec := json.NewDecoder(strings.NewReader(rr.Body.String()))
	dec.UseNumber()
	var raw map[string]any
	require.NoError(t, dec.Decode(&raw))

	total, ok := raw["total_price"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "37", total.String())

	items, ok := raw["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)

	first := items[0].(map[string]any)
	second := items[1].(map[string]any)

	sku, ok := first["sku"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "2", sku.String())
	require.Equal(t, "A", first["name"])
	count, ok := first["count"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "1", count.String())
	price, ok := first["price"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "10", price.String())

	sku, ok = second["sku"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "5", sku.String())
	require.Equal(t, "B", second["name"])
	count, ok = second["count"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "3", count.String())
	price, ok = second["price"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "9", price.String())
}

func TestGetCartEmptyDoesNotCallCatalog(t *testing.T) {
	t.Parallel()

	fake := &fakeCatalog{products: map[int64]model.Product{}}
	h := NewHandler(NewStore(), fake, logger.New())
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/user/3/cart", nil))
	require.Equal(t, http.StatusNotFound, rr.Code)
	require.Zero(t, fake.calls)
}

func TestGetCartBadUser(t *testing.T) {
	t.Parallel()

	h := NewHandler(NewStore(), &fakeCatalog{}, logger.New())
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/user/0/cart", nil))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetCartMissingProduct(t *testing.T) {
	t.Parallel()

	fake := &fakeCatalog{products: map[int64]model.Product{
		1: {SKU: 1, Name: "A", Price: 3},
		2: {SKU: 2, Name: "B", Price: 4},
	}}
	h := NewHandler(NewStore(), fake, logger.New())
	for _, sku := range []string{"1", "2"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/user/9/cart/"+sku, strings.NewReader(`{"count":1}`))
		h.Routes().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	fake.mu.Lock()
	delete(fake.products, 2)
	fake.mu.Unlock()

	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/user/9/cart", nil))
	require.Equal(t, http.StatusPreconditionFailed, rr.Code)
	require.NotContains(t, rr.Body.String(), "total_price")
}

func TestGetCartCatalogDown(t *testing.T) {
	t.Parallel()

	fake := &fakeCatalog{products: map[int64]model.Product{
		1: {SKU: 1, Name: "A", Price: 3},
	}}
	h := NewHandler(NewStore(), fake, logger.New())
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/user/4/cart/1", strings.NewReader(`{"count":1}`)))
	require.Equal(t, http.StatusOK, rr.Code)

	fake.mu.Lock()
	fake.err = status.Error(codes.Unavailable, "down")
	fake.mu.Unlock()

	rr = httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/user/4/cart", nil))
	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.NotContains(t, rr.Body.String(), "total_price")
}

func TestGetCartLoadsProductsInParallel(t *testing.T) {
	t.Parallel()

	gate := &gateCatalog{
		products: map[int64]model.Product{
			1: {SKU: 1, Name: "A", Price: 3},
			2: {SKU: 2, Name: "B", Price: 4},
		},
		need:    2,
		arrived: make(chan struct{}),
		release: make(chan struct{}),
	}
	h := NewHandler(NewStore(), gate, logger.New())
	for _, sku := range []string{"2", "1"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/user/4/cart/"+sku, strings.NewReader(`{"count":1}`))
		h.Routes().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	gate.mu.Lock()
	gate.hold = true
	gate.mu.Unlock()

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rr := httptest.NewRecorder()
		h.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/user/4/cart", nil))
		done <- rr
	}()

	select {
	case <-gate.arrived:
	case <-time.After(time.Second):
		close(gate.release)
		require.FailNow(t, "product calls did not overlap")
	}
	close(gate.release)

	select {
	case rr := <-done:
		require.Equal(t, http.StatusOK, rr.Code)
	case <-time.After(time.Second):
		require.FailNow(t, "get cart did not finish")
	}
}
