package cart

import (
	"testing"

	"homework2/internal/model"
)

func BenchmarkAddToCart(b *testing.B) {
	s := NewStore()
	item := model.CartItem{SKU: 1, Name: "book", Count: 1, Price: 10}
	s.Add(1, item)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(1, item)
	}
}

func BenchmarkGetCart(b *testing.B) {
	s := NewStore()
	for sku := int64(1); sku <= 20; sku++ {
		s.Add(1, model.CartItem{SKU: sku, Name: "book", Count: 2, Price: 10})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items, ok := s.Get(1)
		if !ok || len(items) != 20 {
			b.Fatal("cart")
		}
	}
}
