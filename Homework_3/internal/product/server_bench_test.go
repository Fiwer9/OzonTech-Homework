package product

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "homework2/pkg/api/product"
)

func BenchmarkGetProduct(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "products.json")
	body := []byte(`{"10":{"name":"Book","price":100},"20":{"name":"Pen","price":5}}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		b.Fatal(err)
	}
	store, err := Load(path)
	if err != nil {
		b.Fatal(err)
	}
	srv := NewServer(store)
	req := &pb.GetProductRequest{Sku: 10}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := srv.GetProduct(ctx, req); err != nil {
			b.Fatal(err)
		}
	}
}
