package product

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "homework2/pkg/api/product"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func writeProducts(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "products.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func newServer(t *testing.T) *Server {
	t.Helper()
	path := writeProducts(t, `{"20":{"name":"Pen","price":5},"10":{"name":"Book","price":100}}`)
	store, err := Load(path)
	require.NoError(t, err)
	return NewServer(store)
}

func TestGetProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sku       int64
		wantCode  codes.Code
		wantName  string
		wantPrice int32
	}{
		{name: "found", sku: 10, wantCode: codes.OK, wantName: "Book", wantPrice: 100},
		{name: "zero sku", sku: 0, wantCode: codes.InvalidArgument},
		{name: "negative sku", sku: -3, wantCode: codes.InvalidArgument},
		{name: "missing", sku: 99, wantCode: codes.NotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := newServer(t).GetProduct(context.Background(), &pb.GetProductRequest{Sku: tt.sku})
			if tt.wantCode != codes.OK {
				require.Error(t, err)
				require.Equal(t, tt.wantCode, status.Code(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.sku, resp.GetSku())
			require.Equal(t, tt.wantName, resp.GetName())
			require.Equal(t, tt.wantPrice, resp.GetPrice())
		})
	}
}

func TestListProducts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		page      uint32
		pageSize  uint32
		wantCode  codes.Code
		wantSKUs  []int64
		wantTotal uint32
	}{
		{name: "first page", page: 1, pageSize: 1, wantCode: codes.OK, wantSKUs: []int64{10}, wantTotal: 2},
		{name: "second page", page: 2, pageSize: 1, wantCode: codes.OK, wantSKUs: []int64{20}, wantTotal: 2},
		{name: "whole page", page: 1, pageSize: 10, wantCode: codes.OK, wantSKUs: []int64{10, 20}, wantTotal: 2},
		{name: "past the end", page: 5, pageSize: 1, wantCode: codes.OK, wantSKUs: []int64{}, wantTotal: 2},
		{name: "zero page", page: 0, pageSize: 1, wantCode: codes.InvalidArgument},
		{name: "zero page size", page: 1, pageSize: 0, wantCode: codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := newServer(t).ListProducts(context.Background(), &pb.ListProductsRequest{
				Page:     tt.page,
				PageSize: tt.pageSize,
			})
			if tt.wantCode != codes.OK {
				require.Error(t, err)
				require.Equal(t, tt.wantCode, status.Code(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, resp.GetTotal())
			got := make([]int64, 0, len(resp.GetItems()))
			for _, item := range resp.GetItems() {
				got = append(got, item.GetSku())
			}
			require.Equal(t, tt.wantSKUs, got)
		})
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("skips bad sku", func(t *testing.T) {
		t.Parallel()
		path := writeProducts(t, `{"0":{"name":"no","price":1},"abc":{"name":"bad","price":2},"10":{"name":"ok","price":3}}`)
		store, err := Load(path)
		require.NoError(t, err)
		require.Equal(t, 1, store.Len())
		_, ok := store.Get(10)
		require.True(t, ok)
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
		require.Error(t, err)
	})

	t.Run("broken json", func(t *testing.T) {
		t.Parallel()
		_, err := Load(writeProducts(t, `{`))
		require.Error(t, err)
	})
}
