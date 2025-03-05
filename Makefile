.PHONY: proto build clean update

BINARY_SERVER_NAME=net-perf
BINARY_CLIENT_NAME=net-perfc
BINARY_EXPORTER_NAME=net-perfe
BINARY_DIR=bin

# Generate protobuf code
proto:
	for dir in management measure networkanalysis distributed; do \
		mkdir -p pkg/pb/$$dir; \
		protoc \
			--go_out=. \
			--go_opt=module=github.com/DrC0ns0le/net-perf \
			--go-grpc_out=. \
			--go-grpc_opt=module=github.com/DrC0ns0le/net-perf \
			proto/$$dir/*.proto; \
	done

# Build server, client and exporter
build: proto
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/$(BINARY_SERVER_NAME)-amd64 cmd/daemon/main.go
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_DIR)/$(BINARY_SERVER_NAME)-arm64 cmd/daemon/main.go
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/$(BINARY_CLIENT_NAME)-amd64 cmd/client/main.go
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/$(BINARY_EXPORTER_NAME)-amd64 cmd/exporter/main.go

# Clean generated files and binaries
clean:
	rm -f $(BINARY_DIR)/*
	for dir in management measure networkanalysis; do \
		rm -f proto/$$dir/*.pb.go; \
	done
	

# Update & tidy dependencies
tidy:
	go get -u ./...
	go mod tidy