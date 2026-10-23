package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"

	"homework2/internal/product"
	pb "homework2/pkg/api/product"
	"homework2/pkg/logger"

	"google.golang.org/grpc"
)

func main() {
	addr := flag.String("addr", ":50051", "gRPC listen address")
	data := flag.String("data", "data.json", "path to products json")
	flag.Parse()

	log := logger.New()

	store, err := product.Load(*data)
	if err != nil {
		log.Fatalf("load products: %v", err)
	}
	log.Printf("loaded %d products", store.Len())

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.Printf("product-service listening on %s", lis.Addr())
	if err := run(ctx, lis, store); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func run(ctx context.Context, lis net.Listener, store *product.Store) error {
	srv := grpc.NewServer()
	pb.RegisterProductServiceServer(srv, product.NewServer(store))
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()
	return srv.Serve(lis)
}
