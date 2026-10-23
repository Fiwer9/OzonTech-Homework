//go:build race

package cart

import (
	"sync"
	"testing"

	"homework2/internal/model"

	"github.com/stretchr/testify/require"
)

func TestRaceCart(t *testing.T) {
	t.Parallel()

	s := NewStore()
	const n = 40
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			s.Add(8, model.CartItem{SKU: int64(i%3 + 1), Name: "n", Count: 1, Price: 2})
		}(i)
		go func() {
			defer wg.Done()
			_, _ = s.Get(8)
		}()
	}
	wg.Wait()

	items, ok := s.Get(8)
	require.True(t, ok)
	var sum uint32
	for _, it := range items {
		sum += it.Count
	}
	require.Equal(t, uint32(n), sum)
}
