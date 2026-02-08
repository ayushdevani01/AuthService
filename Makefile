.PHONY: proto docker-up docker-down tidy install-proto-tools

# Install protoc tools
install-proto-tools:
	@echo "Installing protoc and Go plugins..."
	sudo apt update && sudo apt install -y protobuf-compiler golang-go
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Done! Make sure ~/go/bin is in your PATH"

# Generate Go code from proto files
proto:
	@echo "Generating Go code from protos..."
	PATH="$$PATH:$$HOME/go/bin" protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/developer/developer.proto \
		proto/user/user.proto \
		proto/token/token.proto
	@echo "Done!"

# Start all services
docker-up:
	docker-compose up -d

# Stop all services
docker-down:
	docker-compose down

# Clean up volumes
docker-clean:
	docker-compose down -v

# Tidy Go modules
tidy:
	go mod tidy

# Run services
run-developer:
	go run ./services/developer-service

run-user:
	go run ./services/user-service

run-token:
	go run ./services/token-service
