package handler

import (
	"context"
	"errors"

	pb "github.com/ayushdevan01/AuthService/proto/developer"
	"github.com/ayushdevan01/AuthService/services/developer-service/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeveloperHandler struct {
	pb.UnimplementedDeveloperServiceServer
	service *service.DeveloperService
}

func NewDeveloperHandler(svc *service.DeveloperService) *DeveloperHandler {
	return &DeveloperHandler{service: svc}
}

func (h *DeveloperHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password, and name are required")
	}

	result, err := h.service.Register(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Error(codes.Internal, "failed to register developer")
	}

	return &pb.RegisterResponse{
		Developer: &pb.Developer{
			Id:        result.Developer.ID,
			Email:     result.Developer.Email,
			Name:      result.Developer.Name,
			CreatedAt: timestamppb.New(result.Developer.CreatedAt),
			UpdatedAt: timestamppb.New(result.Developer.UpdatedAt),
		},
		AccessToken: result.AccessToken,
	}, nil
}

func (h *DeveloperHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	result, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Error(codes.Internal, "failed to login")
	}

	return &pb.LoginResponse{
		Developer: &pb.Developer{
			Id:        result.Developer.ID,
			Email:     result.Developer.Email,
			Name:      result.Developer.Name,
			CreatedAt: timestamppb.New(result.Developer.CreatedAt),
			UpdatedAt: timestamppb.New(result.Developer.UpdatedAt),
		},
		AccessToken: result.AccessToken,
	}, nil
}

func (h *DeveloperHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	// Client just discards the token
	return &pb.LogoutResponse{Success: true}, nil
}

func (h *DeveloperHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	if req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "developer_id is required")
	}

	dev, err := h.service.GetProfile(ctx, req.DeveloperId)
	if err != nil {
		if errors.Is(err, service.ErrDeveloperNotFound) {
			return &pb.GetProfileResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get profile")
	}

	return &pb.GetProfileResponse{
		Developer: &pb.Developer{
			Id:        dev.ID,
			Email:     dev.Email,
			Name:      dev.Name,
			CreatedAt: timestamppb.New(dev.CreatedAt),
			UpdatedAt: timestamppb.New(dev.UpdatedAt),
		},
		Found: true,
	}, nil
}

func (h *DeveloperHandler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	if req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "developer_id is required")
	}

	var name, password *string
	if req.Name != nil {
		name = req.Name
	}
	if req.Password != nil {
		password = req.Password
	}

	dev, err := h.service.UpdateProfile(ctx, req.DeveloperId, name, password)
	if err != nil {
		if errors.Is(err, service.ErrDeveloperNotFound) {
			return nil, status.Error(codes.NotFound, "developer not found")
		}
		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	return &pb.UpdateProfileResponse{
		Developer: &pb.Developer{
			Id:        dev.ID,
			Email:     dev.Email,
			Name:      dev.Name,
			CreatedAt: timestamppb.New(dev.CreatedAt),
			UpdatedAt: timestamppb.New(dev.UpdatedAt),
		},
	}, nil
}
