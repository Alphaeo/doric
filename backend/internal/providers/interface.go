package providers

import "context"

type VPS struct {
	ID       string
	IP       string
	Provider string
}

type Provider interface {
	CreateVPS(ctx context.Context, name string) (*VPS, error)
	GetVPS(ctx context.Context, id string) (*VPS, error)
}
