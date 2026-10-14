package cart

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	_ "homework2/docs"
	"homework2/internal/model"
	"homework2/pkg/logger"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AddItemRequest struct {
	Count uint32 `json:"count" example:"12"`
}

type Handler struct {
	store  *Store
	client *ProductClient
	log    *logger.Logger
}

func NewHandler(store *Store, client *ProductClient, log *logger.Logger) *Handler {
	return &Handler{store: store, client: client, log: log}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(h.logging)
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
	r.Post("/user/{user_id}/cart/{sku_id}", h.addItem)
	r.Get("/user/{user_id}/cart", h.getCart)
	return r
}

// @Summary Добавить товар в корзину
// @Tags cart
// @Accept json
// @Produce json
// @Param user_id path int true "Идентификатор пользователя" example(1007)
// @Param sku_id path int true "Идентификатор товара" example(1076963)
// @Param request body AddItemRequest true "Количество товара"
// @Success 200
// @Failure 400 {string} string
// @Failure 412 {string} string
// @Failure 500 {string} string
// @Router /user/{user_id}/cart/{sku_id} [post]
func (h *Handler) addItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseID(chi.URLParam(r, "user_id"))
	if !ok {
		http.Error(w, "user id must be a natural number", http.StatusBadRequest)
		return
	}
	sku, ok := parseID(chi.URLParam(r, "sku_id"))
	if !ok {
		http.Error(w, "sku must be a natural number", http.StatusBadRequest)
		return
	}

	var body AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Count == 0 {
		http.Error(w, "count must be a natural number", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	p, err := h.client.GetProduct(ctx, sku)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			http.Error(w, "sku must exist in product-service", http.StatusPreconditionFailed)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.store.Add(userID, model.CartItem{
		SKU:   p.SKU,
		Name:  p.Name,
		Count: body.Count,
		Price: p.Price,
	})
	w.WriteHeader(http.StatusOK)
}

// @Summary Получить содержимое корзины
// @Tags cart
// @Produce json
// @Param user_id path int true "Идентификатор пользователя" example(1007)
// @Success 200 {object} model.Cart
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 500 {string} string
// @Router /user/{user_id}/cart [get]
func (h *Handler) getCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseID(chi.URLParam(r, "user_id"))
	if !ok {
		http.Error(w, "user id must be a natural number", http.StatusBadRequest)
		return
	}

	items, ok := h.store.Get(userID)
	if !ok {
		http.Error(w, "cart not found", http.StatusNotFound)
		return
	}

	var total uint32
	for _, it := range items {
		total += it.Price * it.Count
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(model.Cart{
		Items:      items,
		TotalPrice: total,
	})
}

func parseID(s string) (int64, bool) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (h *Handler) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		h.log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.code, time.Since(start))
	})
}
