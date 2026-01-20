# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o doric-agent main.go

# Run stage
FROM alpine:latest

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/doric-agent .

# Agent might need Docker-in-Docker or access to host docker socket
# For now, just the basic binary execution
CMD ["./doric-agent"]
