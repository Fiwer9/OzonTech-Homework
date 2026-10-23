package cart

import (
	"sort"
	"sync"
	"sync/atomic"

	"homework2/internal/model"
)

type storedItem struct {
	name  string
	price uint32
	count atomic.Uint32
}

type Store struct {
	mu    sync.RWMutex
	carts map[int64]map[int64]*storedItem
}

func NewStore() *Store {
	return &Store{carts: make(map[int64]map[int64]*storedItem)}
}

func (s *Store) Add(userID int64, item model.CartItem) {
	if s.bump(userID, item.SKU, item.Count) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userCart, ok := s.carts[userID]
	if !ok {
		userCart = make(map[int64]*storedItem)
		s.carts[userID] = userCart
	}
	if cur, ok := userCart[item.SKU]; ok {
		cur.count.Add(item.Count)
		return
	}
	created := &storedItem{name: item.Name, price: item.Price}
	created.count.Store(item.Count)
	userCart[item.SKU] = created
}

func (s *Store) bump(userID, sku int64, count uint32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userCart, ok := s.carts[userID]
	if !ok {
		return false
	}
	cur, ok := userCart[sku]
	if !ok {
		return false
	}
	cur.count.Add(count)
	return true
}

func (s *Store) Get(userID int64) ([]model.CartItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userCart, ok := s.carts[userID]
	if !ok || len(userCart) == 0 {
		return nil, false
	}

	items := make([]model.CartItem, 0, len(userCart))
	for sku, it := range userCart {
		items = append(items, model.CartItem{
			SKU:   sku,
			Name:  it.name,
			Count: it.count.Load(),
			Price: it.price,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SKU < items[j].SKU })
	return items, true
}
