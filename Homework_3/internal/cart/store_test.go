package cart

import (
	"testing"

	"homework2/internal/model"

	"github.com/stretchr/testify/require"
)

func TestStoreAddAndGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		adds []model.CartItem
		want []model.CartItem
	}{
		{
			name: "same sku is summed and items are sorted",
			adds: []model.CartItem{
				{SKU: 5, Name: "b", Count: 1, Price: 10},
				{SKU: 2, Name: "a", Count: 3, Price: 4},
				{SKU: 5, Name: "b", Count: 2, Price: 10},
			},
			want: []model.CartItem{
				{SKU: 2, Name: "a", Count: 3, Price: 4},
				{SKU: 5, Name: "b", Count: 3, Price: 10},
			},
		},
		{
			name: "single item",
			adds: []model.CartItem{{SKU: 8, Name: "pen", Count: 4, Price: 15}},
			want: []model.CartItem{{SKU: 8, Name: "pen", Count: 4, Price: 15}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			for _, it := range tt.adds {
				s.Add(42, it)
			}
			got, ok := s.Get(42)
			require.True(t, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStoreEmptyCart(t *testing.T) {
	t.Parallel()

	s := NewStore()
	_, ok := s.Get(1)
	require.False(t, ok)

	s.Add(1, model.CartItem{SKU: 3, Name: "a", Count: 1, Price: 1})
	_, ok = s.Get(2)
	require.False(t, ok)
}
