package product

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"

	"homework2/internal/model"
)

type Store struct {
	mu    sync.RWMutex
	bySKU map[int64]model.Product
	all   []model.Product
}

func Load(path string) (*Store, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open products: %w", err)
	}
	defer f.Close()

	var raw map[string]struct {
		Name  string `json:"name"`
		Price uint32 `json:"price"`
	}
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode products: %w", err)
	}

	s := &Store{
		bySKU: make(map[int64]model.Product, len(raw)),
		all:   make([]model.Product, 0, len(raw)),
	}
	for k, v := range raw {
		sku, err := strconv.ParseInt(k, 10, 64)
		if err != nil || sku <= 0 {
			continue
		}
		p := model.Product{SKU: sku, Name: v.Name, Price: v.Price}
		s.bySKU[sku] = p
		s.all = append(s.all, p)
	}
	sort.Slice(s.all, func(i, j int) bool { return s.all[i].SKU < s.all[j].SKU })
	return s, nil
}

func (s *Store) Get(sku int64) (model.Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.bySKU[sku]
	return p, ok
}

func (s *Store) List(page, pageSize uint32) ([]model.Product, uint32) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := uint32(len(s.all))
	start := (page - 1) * pageSize
	if start >= total {
		return nil, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	out := make([]model.Product, end-start)
	copy(out, s.all[start:end])
	return out, total
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.all)
}
