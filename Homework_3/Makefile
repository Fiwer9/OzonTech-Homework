.PHONY: generate run-all test test-race test-api bench

generate:
	protoc -I proto \
		--go_out=. --go_opt=module=homework2 \
		--go-grpc_out=. --go-grpc_opt=module=homework2 \
		proto/product/product.proto

run-all:
	go run ./cmd/product-service &
	sleep 2
	go run ./cmd/cart

test:
	go test ./...

test-race:
	go test -race ./...

test-api:
	go test -v -count=1 -tags=api ./test/

bench:
	go test -bench=. -benchmem -run=^$$ ./internal/cart ./internal/product
