package cart

import (
	"context"
	"fmt"

	"homework2/internal/model"
	pb "homework2/pkg/api/product"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProductClient struct {
	client pb.ProductServiceClient
	conn   *grpc.ClientConn
}

func NewProductClient(addr string) (*ProductClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial product-service: %w", err)
	}
	return &ProductClient{
		client: pb.NewProductServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *ProductClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *ProductClient) GetProduct(ctx context.Context, sku int64) (model.Product, error) {
	resp, err := c.client.GetProduct(ctx, &pb.GetProductRequest{Sku: sku})
	if err != nil {
		return model.Product{}, err
	}
	return model.Product{
		SKU:   resp.GetSku(),
		Name:  resp.GetName(),
		Price: uint32(resp.GetPrice()),
	}, nil
}
