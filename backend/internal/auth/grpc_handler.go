package auth

import (
	"context"
	"doric/backend/internal/db"
	pb "doric/backend/pkg/proto/doric"
	"fmt"
)

type GrpcHandler struct {
	pb.UnimplementedAuthServiceServer
	authSvc *AuthService
	userRepo *db.UserRepo
}

func NewGrpcHandler(authSvc *AuthService, userRepo *db.UserRepo) *GrpcHandler {
	return &GrpcHandler{
		authSvc:  authSvc,
		userRepo: userRepo,
	}
}

func (h *GrpcHandler) GetAuthUrl(ctx context.Context, req *pb.GetAuthUrlRequest) (*pb.GetAuthUrlResponse, error) {
	url := h.authSvc.GetAuthURL()
	return &pb.GetAuthUrlResponse{Url: url}, nil
}

func (h *GrpcHandler) ExchangeCode(ctx context.Context, req *pb.ExchangeCodeRequest) (*pb.AuthResponse, error) {
	userInfo, err := h.authSvc.ExchangeCode(ctx, req.Code)
	if err != nil {
		return &pb.AuthResponse{Success: false, Error: err.Error()}, nil
	}

	user := &db.User{
		ID:      userInfo.Sub,
		Email:   userInfo.Email,
		Name:    userInfo.Name,
		Picture: userInfo.Picture,
	}

	if err := h.userRepo.UpsertUser(ctx, user); err != nil {
		return &pb.AuthResponse{Success: false, Error: fmt.Sprintf("failed to save user: %v", err)}, nil
	}

	return &pb.AuthResponse{
		Success: true,
		User: &pb.User{
			Id:      user.ID,
			Email:   user.Email,
			Name:    user.Name,
			Picture: user.Picture,
		},
	}, nil
}
