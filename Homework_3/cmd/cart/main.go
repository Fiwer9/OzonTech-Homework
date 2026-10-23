package main

import (
	"context"
	"errors"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

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

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	h := cart.NewHandler(cart.NewStore(), client, log)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.Printf("cart listening on %s", lis.Addr())
	if err := run(ctx, lis, h.Routes()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func run(ctx context.Context, lis net.Listener, handler http.Handler) error {
	srv := &http.Server{Handler: handler}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	err := srv.Serve(lis)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
