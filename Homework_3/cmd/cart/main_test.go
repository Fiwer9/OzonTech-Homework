package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"homework2/internal/cart"
	"homework2/internal/model"
	"homework2/pkg/logger"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubCatalog struct{}

func (stubCatalog) GetProduct(_ context.Context, sku int64) (model.Product, error) {
	if sku != 5 {
		return model.Product{}, status.Error(codes.NotFound, "product not found")
	}
	return model.Product{SKU: 5, Name: "Pen", Price: 12}, nil
}

func TestRunCart(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	h := cart.NewHandler(cart.NewStore(), stubCatalog{}, logger.New())
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- run(ctx, lis, h.Routes()) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-errCh)
	})

	base := "http://" + lis.Addr().String()
	require.Eventually(t, func() bool {
		resp, err := http.Post(base+"/user/1/cart/5", "application/json", strings.NewReader(`{"count":2}`))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 20*time.Millisecond)

	resp, err := http.Get(base + "/user/1/cart")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body model.Cart
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Len(t, body.Items, 1)
	require.Equal(t, "Pen", body.Items[0].Name)
	require.Equal(t, uint32(2), body.Items[0].Count)
	require.Equal(t, uint32(24), body.TotalPrice)
}
