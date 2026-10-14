package main

import (
	"flag"
	"net/http"

	"homework2/internal/cart"
	"homework2/pkg/logger"
)

// @title Cart API
// @version 1.0
// @description Сервис корзины пользователя
// @host localhost:8080
// @BasePath /
func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	productAddr := flag.String("product-addr", "localhost:50051", "product-service gRPC address")
	flag.Parse()

	log := logger.New()

	client, err := cart.NewProductClient(*productAddr)
	if err != nil {
		log.Fatalf("product client: %v", err)
	}
	defer client.Close()

	h := cart.NewHandler(cart.NewStore(), client, log)
	log.Printf("cart listening on %s", *addr)
	if err := http.ListenAndServe(*addr, h.Routes()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
