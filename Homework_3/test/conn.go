//go:build api

package apitest

import (
	"net/http"
	"testing"
	"time"

	pb "homework2/pkg/api/product"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func productClient(t *testing.T) pb.ProductServiceClient {
	t.Helper()
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewProductServiceClient(conn)
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 3 * time.Second}
}
