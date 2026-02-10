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

type OAuthProviderHandler struct {
	pb.UnimplementedDeveloperServiceServer
	appService *service.AppService
}

func NewOAuthProviderHandler(appService *service.AppService) *OAuthProviderHandler {
	return &OAuthProviderHandler{appService: appService}
}

func (h *OAuthProviderHandler) AddOAuthProvider(ctx context.Context, req *pb.AddOAuthProviderRequest) (*pb.AddOAuthProviderResponse, error) {
	if req.AppId == "" || req.DeveloperId == "" || req.Provider == "" || req.ClientId == "" || req.ClientSecret == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id, developer_id, provider, client_id, and client_secret are required")
	}

	provider, err := h.appService.AddOAuthProvider(ctx, req.AppId, req.DeveloperId, req.Provider, req.ClientId, req.ClientSecret, req.Scopes)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to add oauth provider")
	}

	return &pb.AddOAuthProviderResponse{
		Provider: &pb.OAuthProvider{
			Id:        provider.ID,
			AppId:     provider.AppID,
			Provider:  provider.Provider,
			ClientId:  provider.ClientID,
			Scopes:    provider.Scopes,
			Enabled:   provider.Enabled,
			CreatedAt: timestamppb.New(provider.CreatedAt),
		},
	}, nil
}

func (h *OAuthProviderHandler) GetOAuthProvider(ctx context.Context, req *pb.GetOAuthProviderRequest) (*pb.GetOAuthProviderResponse, error) {
	if req.AppId == "" || req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id and provider are required")
	}

	result, err := h.appService.GetOAuthProvider(ctx, req.AppId, req.Provider)
	if err != nil {
		if errors.Is(err, service.ErrProviderNotFound) {
			return &pb.GetOAuthProviderResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get oauth provider")
	}

	return &pb.GetOAuthProviderResponse{
		Provider: &pb.OAuthProvider{
			Id:        result.Provider.ID,
			AppId:     result.Provider.AppID,
			Provider:  result.Provider.Provider,
			ClientId:  result.Provider.ClientID,
			Scopes:    result.Provider.Scopes,
			Enabled:   result.Provider.Enabled,
			CreatedAt: timestamppb.New(result.Provider.CreatedAt),
		},
		ClientSecret: result.ClientSecret,
		Found:        true,
	}, nil
}

func (h *OAuthProviderHandler) ListOAuthProviders(ctx context.Context, req *pb.ListOAuthProvidersRequest) (*pb.ListOAuthProvidersResponse, error) {
	if req.AppId == "" || req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id and developer_id are required")
	}

	providers, err := h.appService.ListOAuthProviders(ctx, req.AppId, req.DeveloperId)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to list oauth providers")
	}

	var pbProviders []*pb.OAuthProvider
	for _, p := range providers {
		pbProviders = append(pbProviders, &pb.OAuthProvider{
			Id:        p.ID,
			AppId:     p.AppID,
			Provider:  p.Provider,
			ClientId:  p.ClientID,
			Scopes:    p.Scopes,
			Enabled:   p.Enabled,
			CreatedAt: timestamppb.New(p.CreatedAt),
		})
	}

	return &pb.ListOAuthProvidersResponse{Providers: pbProviders}, nil
}

func (h *OAuthProviderHandler) UpdateOAuthProvider(ctx context.Context, req *pb.UpdateOAuthProviderRequest) (*pb.UpdateOAuthProviderResponse, error) {
	if req.AppId == "" || req.DeveloperId == "" || req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id, developer_id, and provider are required")
	}

	provider, err := h.appService.UpdateOAuthProvider(ctx, req.AppId, req.DeveloperId, req.Provider, req.ClientId, req.ClientSecret, req.Scopes, req.Enabled)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to update oauth provider")
	}

	return &pb.UpdateOAuthProviderResponse{
		Provider: &pb.OAuthProvider{
			Id:        provider.ID,
			AppId:     provider.AppID,
			Provider:  provider.Provider,
			ClientId:  provider.ClientID,
			Scopes:    provider.Scopes,
			Enabled:   provider.Enabled,
			CreatedAt: timestamppb.New(provider.CreatedAt),
		},
	}, nil
}

func (h *OAuthProviderHandler) DeleteOAuthProvider(ctx context.Context, req *pb.DeleteOAuthProviderRequest) (*pb.DeleteOAuthProviderResponse, error) {
	if req.AppId == "" || req.DeveloperId == "" || req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id, developer_id, and provider are required")
	}

	err := h.appService.DeleteOAuthProvider(ctx, req.AppId, req.DeveloperId, req.Provider)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to delete oauth provider")
	}

	return &pb.DeleteOAuthProviderResponse{Success: true}, nil
}
