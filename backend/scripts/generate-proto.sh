#!/bin/sh
set -e

echo "Installing protoc and Go plugins..."
apk add --no-cache protobuf protobuf-dev

echo "Installing Go protoc plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo "Generating proto files..."
mkdir -p pkg/proto/doric
protoc --go_out=pkg/proto --go_opt=paths=source_relative \
       --go-grpc_out=pkg/proto --go-grpc_opt=paths=source_relative \
       proto/doric.proto

echo "Proto generation complete!"
