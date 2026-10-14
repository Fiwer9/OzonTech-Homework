.PHONY: generate run-all

generate:
	protoc -I proto \
		--go_out=. --go_opt=module=homework2 \
		--go-grpc_out=. --go-grpc_opt=module=homework2 \
		proto/product/product.proto

run-all:
	go run ./cmd/product-service &
	sleep 2
	go run ./cmd/cart
