package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"homework2/internal/product"
	pb "homework2/pkg/api/product"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestRunProduct(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "products.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"10":{"name":"Book","price":100}}`), 0o644))
	store, err := product.Load(path)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- run(ctx, lis, store) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-errCh)
	})

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pb.NewProductServiceClient(conn)

	require.Eventually(t, func() bool {
		resp, err := client.GetProduct(context.Background(), &pb.GetProductRequest{Sku: 10})
		return err == nil && resp.GetName() == "Book" && resp.GetPrice() == 100
	}, 2*time.Second, 20*time.Millisecond)
}
