FROM golang:1.21-alpine AS builder

WORKDIR /app

# git pour les clones, curl pour healthcheck
RUN apk add --no-cache git

COPY go.mod ./
RUN go mod download

COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o doric-agent main.go

# ── Runtime ────────────────────────────────────────────────────────────────
FROM alpine:latest

# git (clone repos), docker CLI (build/run containers)
RUN apk add --no-cache git docker-cli

WORKDIR /root/
COPY --from=builder /app/doric-agent .

EXPOSE 50052
CMD ["./doric-agent"]
