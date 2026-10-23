package product

import (
	"context"

	pb "homework2/pkg/api/product"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedProductServiceServer
	store *Store
}

func NewServer(store *Store) *Server {
	return &Server{store: store}
}

func (s *Server) GetProduct(_ context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	if req.GetSku() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "sku must be a natural number")
	}
	p, ok := s.store.Get(req.GetSku())
	if !ok {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	return &pb.GetProductResponse{
		Sku:   p.SKU,
		Name:  p.Name,
		Price: int32(p.Price),
	}, nil
}

func (s *Server) ListProducts(_ context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	if req.GetPage() == 0 || req.GetPageSize() == 0 {
		return nil, status.Error(codes.InvalidArgument, "page and page_size must be greater than zero")
	}
	items, total := s.store.List(req.GetPage(), req.GetPageSize())
	resp := &pb.ListProductsResponse{
		Items: make([]*pb.GetProductResponse, 0, len(items)),
		Total: total,
	}
	for _, p := range items {
		resp.Items = append(resp.Items, &pb.GetProductResponse{
			Sku:   p.SKU,
			Name:  p.Name,
			Price: int32(p.Price),
		})
	}
	return resp, nil
}
