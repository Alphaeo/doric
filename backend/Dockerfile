# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies and protoc
RUN apk add --no-cache git protobuf protobuf-dev && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Copy only what's needed for proto generation
COPY proto/ ./proto/
COPY backend/go.mod ./backend/

WORKDIR /app/backend
# Initialize a clean go.mod if necessary, but we have one
RUN go mod download

# Copy the rest of the backend source
COPY backend/ ./

# Generate proto files
RUN mkdir -p pkg/proto && \
    protoc --proto_path=/app/proto --go_out=. --go_opt=module=doric/backend \
    --go-grpc_out=. --go-grpc_opt=module=doric/backend \
    /app/proto/doric.proto

# Build
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /doric-backend ./cmd/api

# Run stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /doric-backend .
EXPOSE 50051
CMD ["./doric-backend"]
