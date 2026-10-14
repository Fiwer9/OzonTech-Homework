package main

import (
	"flag"
	"net"

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

	s := grpc.NewServer()
	pb.RegisterProductServiceServer(s, product.NewServer(store))
	log.Printf("product-service listening on %s", *addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
