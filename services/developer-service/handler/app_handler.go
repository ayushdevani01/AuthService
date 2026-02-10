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

type AppHandler struct {
	pb.UnimplementedDeveloperServiceServer
	appService *service.AppService
}

func NewAppHandler(appService *service.AppService) *AppHandler {
	return &AppHandler{appService: appService}
}

func (h *AppHandler) CreateApp(ctx context.Context, req *pb.CreateAppRequest) (*pb.CreateAppResponse, error) {
	if req.DeveloperId == "" || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "developer_id and name are required")
	}

	result, err := h.appService.CreateApp(ctx, req.DeveloperId, req.Name, req.LogoUrl, req.RedirectUrls)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create app")
	}

	return &pb.CreateAppResponse{
		App: &pb.App{
			Id:           result.App.ID,
			DeveloperId:  result.App.DeveloperID,
			Name:         result.App.Name,
			LogoUrl:      result.App.LogoURL,
			RedirectUrls: result.App.RedirectURLs,
			CreatedAt:    timestamppb.New(result.App.CreatedAt),
			UpdatedAt:    timestamppb.New(result.App.UpdatedAt),
		},
		ApiKey: result.APIKey,
		SigningKey: &pb.SigningKey{
			Id:        result.SigningKey.ID,
			AppId:     result.SigningKey.AppID,
			Kid:       result.SigningKey.KID,
			PublicKey: result.SigningKey.PublicKey,
			IsActive:  result.SigningKey.IsActive,
			CreatedAt: timestamppb.New(result.SigningKey.CreatedAt),
		},
	}, nil
}

func (h *AppHandler) GetApp(ctx context.Context, req *pb.GetAppRequest) (*pb.GetAppResponse, error) {
	if req.AppId == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	app, err := h.appService.GetApp(ctx, req.AppId)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return &pb.GetAppResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get app")
	}

	return &pb.GetAppResponse{
		App: &pb.App{
			Id:           app.ID,
			DeveloperId:  app.DeveloperID,
			Name:         app.Name,
			LogoUrl:      app.LogoURL,
			RedirectUrls: app.RedirectURLs,
			CreatedAt:    timestamppb.New(app.CreatedAt),
			UpdatedAt:    timestamppb.New(app.UpdatedAt),
		},
		Found: true,
	}, nil
}

func (h *AppHandler) ListApps(ctx context.Context, req *pb.ListAppsRequest) (*pb.ListAppsResponse, error) {
	if req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "developer_id is required")
	}

	apps, err := h.appService.ListApps(ctx, req.DeveloperId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list apps")
	}

	var pbApps []*pb.App
	for _, app := range apps {
		pbApps = append(pbApps, &pb.App{
			Id:           app.ID,
			DeveloperId:  app.DeveloperID,
			Name:         app.Name,
			LogoUrl:      app.LogoURL,
			RedirectUrls: app.RedirectURLs,
			CreatedAt:    timestamppb.New(app.CreatedAt),
			UpdatedAt:    timestamppb.New(app.UpdatedAt),
		})
	}

	return &pb.ListAppsResponse{Apps: pbApps}, nil
}

func (h *AppHandler) UpdateApp(ctx context.Context, req *pb.UpdateAppRequest) (*pb.UpdateAppResponse, error) {
	if req.Id == "" || req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "id and developer_id are required")
	}

	app, err := h.appService.UpdateApp(ctx, req.Id, req.DeveloperId, req.Name, req.LogoUrl, req.RedirectUrls)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to update app")
	}

	return &pb.UpdateAppResponse{
		App: &pb.App{
			Id:           app.ID,
			DeveloperId:  app.DeveloperID,
			Name:         app.Name,
			LogoUrl:      app.LogoURL,
			RedirectUrls: app.RedirectURLs,
			CreatedAt:    timestamppb.New(app.CreatedAt),
			UpdatedAt:    timestamppb.New(app.UpdatedAt),
		},
	}, nil
}

func (h *AppHandler) DeleteApp(ctx context.Context, req *pb.DeleteAppRequest) (*pb.DeleteAppResponse, error) {
	if req.Id == "" || req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "id and developer_id are required")
	}

	err := h.appService.DeleteApp(ctx, req.Id, req.DeveloperId)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to delete app")
	}

	return &pb.DeleteAppResponse{Success: true}, nil
}

func (h *AppHandler) RotateApiKey(ctx context.Context, req *pb.RotateApiKeyRequest) (*pb.RotateApiKeyResponse, error) {
	if req.AppId == "" || req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id and developer_id are required")
	}

	newKey, err := h.appService.RotateAPIKey(ctx, req.AppId, req.DeveloperId)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to rotate api key")
	}

	return &pb.RotateApiKeyResponse{NewApiKey: newKey}, nil
}

func (h *AppHandler) RotateSigningKeys(ctx context.Context, req *pb.RotateSigningKeysRequest) (*pb.RotateSigningKeysResponse, error) {
	if req.AppId == "" || req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id and developer_id are required")
	}

	newKey, oldKey, err := h.appService.RotateSigningKeys(ctx, req.AppId, req.DeveloperId, int(req.GracePeriodHours))
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to rotate signing keys")
	}

	resp := &pb.RotateSigningKeysResponse{
		NewKey: &pb.SigningKey{
			Id:        newKey.ID,
			AppId:     newKey.AppID,
			Kid:       newKey.KID,
			PublicKey: newKey.PublicKey,
			IsActive:  newKey.IsActive,
			CreatedAt: timestamppb.New(newKey.CreatedAt),
		},
	}

	if oldKey != nil {
		resp.OldKey = &pb.SigningKey{
			Id:        oldKey.ID,
			AppId:     oldKey.AppID,
			Kid:       oldKey.KID,
			PublicKey: oldKey.PublicKey,
			IsActive:  oldKey.IsActive,
			CreatedAt: timestamppb.New(oldKey.CreatedAt),
		}
		if oldKey.ExpiresAt != nil {
			resp.OldKey.ExpiresAt = timestamppb.New(*oldKey.ExpiresAt)
		}
		if oldKey.RotatedAt != nil {
			resp.OldKey.RotatedAt = timestamppb.New(*oldKey.RotatedAt)
		}
	}

	return resp, nil
}

func (h *AppHandler) ListSigningKeys(ctx context.Context, req *pb.ListSigningKeysRequest) (*pb.ListSigningKeysResponse, error) {
	if req.AppId == "" || req.DeveloperId == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id and developer_id are required")
	}

	keys, err := h.appService.ListSigningKeys(ctx, req.AppId, req.DeveloperId, req.IncludeExpired)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		if errors.Is(err, service.ErrNotAppOwner) {
			return nil, status.Error(codes.PermissionDenied, "not the owner of this app")
		}
		return nil, status.Error(codes.Internal, "failed to list signing keys")
	}

	var pbKeys []*pb.SigningKey
	for _, key := range keys {
		pbKey := &pb.SigningKey{
			Id:        key.ID,
			AppId:     key.AppID,
			Kid:       key.KID,
			PublicKey: key.PublicKey,
			IsActive:  key.IsActive,
			CreatedAt: timestamppb.New(key.CreatedAt),
		}
		if key.ExpiresAt != nil {
			pbKey.ExpiresAt = timestamppb.New(*key.ExpiresAt)
		}
		if key.RotatedAt != nil {
			pbKey.RotatedAt = timestamppb.New(*key.RotatedAt)
		}
		pbKeys = append(pbKeys, pbKey)
	}

	return &pb.ListSigningKeysResponse{Keys: pbKeys}, nil
}

func (h *AppHandler) GetActiveSigningKey(ctx context.Context, req *pb.GetActiveSigningKeyRequest) (*pb.GetActiveSigningKeyResponse, error) {
	if req.AppId == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	result, err := h.appService.GetActiveSigningKey(ctx, req.AppId)
	if err != nil {
		if errors.Is(err, service.ErrAppNotFound) {
			return &pb.GetActiveSigningKeyResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get active signing key")
	}

	return &pb.GetActiveSigningKeyResponse{
		Key: &pb.SigningKey{
			Id:        result.Key.ID,
			AppId:     result.Key.AppID,
			Kid:       result.Key.KID,
			PublicKey: result.Key.PublicKey,
			IsActive:  result.Key.IsActive,
			CreatedAt: timestamppb.New(result.Key.CreatedAt),
		},
		PrivateKey: result.PrivateKey,
		Found:      true,
	}, nil
}
