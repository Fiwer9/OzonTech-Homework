package cart

import (
	"sort"
	"sync"

	"homework2/internal/model"
)

type Store struct {
	mu    sync.Mutex
	carts map[int64]map[int64]model.CartItem
}

func NewStore() *Store {
	return &Store{carts: make(map[int64]map[int64]model.CartItem)}
}

func (s *Store) Add(userID int64, item model.CartItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userCart, ok := s.carts[userID]
	if !ok {
		userCart = make(map[int64]model.CartItem)
		s.carts[userID] = userCart
	}
	cur, ok := userCart[item.SKU]
	if ok {
		item.Count += cur.Count
	}
	userCart[item.SKU] = item
}

func (s *Store) Get(userID int64) ([]model.CartItem, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userCart, ok := s.carts[userID]
	if !ok || len(userCart) == 0 {
		return nil, false
	}

	items := make([]model.CartItem, 0, len(userCart))
	for _, it := range userCart {
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SKU < items[j].SKU })
	return items, true
}
