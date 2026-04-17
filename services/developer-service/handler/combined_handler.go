package handler

import (
	"context"

	pb "github.com/ayushdevan01/AuthService/proto/developer"
	"github.com/ayushdevan01/AuthService/services/developer-service/service"
)

type CombinedHandler struct {
	pb.UnimplementedDeveloperServiceServer
	*DeveloperHandler
	*AppHandler
	*OAuthProviderHandler
}

func NewCombinedHandler(developerService *service.DeveloperService, appService *service.AppService) *CombinedHandler {
	return &CombinedHandler{
		DeveloperHandler:     NewDeveloperHandler(developerService),
		AppHandler:           NewAppHandler(appService),
		OAuthProviderHandler: NewOAuthProviderHandler(appService),
	}
}

func (h *CombinedHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return h.DeveloperHandler.Register(ctx, req)
}

func (h *CombinedHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return h.DeveloperHandler.Login(ctx, req)
}

func (h *CombinedHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return h.DeveloperHandler.Logout(ctx, req)
}

func (h *CombinedHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return h.DeveloperHandler.GetProfile(ctx, req)
}

func (h *CombinedHandler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	return h.DeveloperHandler.UpdateProfile(ctx, req)
}

func (h *CombinedHandler) CreateApp(ctx context.Context, req *pb.CreateAppRequest) (*pb.CreateAppResponse, error) {
	return h.AppHandler.CreateApp(ctx, req)
}

func (h *CombinedHandler) GetApp(ctx context.Context, req *pb.GetAppRequest) (*pb.GetAppResponse, error) {
	return h.AppHandler.GetApp(ctx, req)
}

func (h *CombinedHandler) GetPublicApp(ctx context.Context, req *pb.GetPublicAppRequest) (*pb.GetPublicAppResponse, error) {
	return h.AppHandler.GetPublicApp(ctx, req)
}

func (h *CombinedHandler) ListApps(ctx context.Context, req *pb.ListAppsRequest) (*pb.ListAppsResponse, error) {
	return h.AppHandler.ListApps(ctx, req)
}

func (h *CombinedHandler) UpdateApp(ctx context.Context, req *pb.UpdateAppRequest) (*pb.UpdateAppResponse, error) {
	return h.AppHandler.UpdateApp(ctx, req)
}

func (h *CombinedHandler) DeleteApp(ctx context.Context, req *pb.DeleteAppRequest) (*pb.DeleteAppResponse, error) {
	return h.AppHandler.DeleteApp(ctx, req)
}

func (h *CombinedHandler) VerifyApiKey(ctx context.Context, req *pb.VerifyApiKeyRequest) (*pb.VerifyApiKeyResponse, error) {
	return h.AppHandler.VerifyApiKey(ctx, req)
}

func (h *CombinedHandler) RotateApiKey(ctx context.Context, req *pb.RotateApiKeyRequest) (*pb.RotateApiKeyResponse, error) {
	return h.AppHandler.RotateApiKey(ctx, req)
}

func (h *CombinedHandler) RotateSigningKeys(ctx context.Context, req *pb.RotateSigningKeysRequest) (*pb.RotateSigningKeysResponse, error) {
	return h.AppHandler.RotateSigningKeys(ctx, req)
}

func (h *CombinedHandler) ListSigningKeys(ctx context.Context, req *pb.ListSigningKeysRequest) (*pb.ListSigningKeysResponse, error) {
	return h.AppHandler.ListSigningKeys(ctx, req)
}

func (h *CombinedHandler) GetActiveSigningKey(ctx context.Context, req *pb.GetActiveSigningKeyRequest) (*pb.GetActiveSigningKeyResponse, error) {
	return h.AppHandler.GetActiveSigningKey(ctx, req)
}

func (h *CombinedHandler) AddOAuthProvider(ctx context.Context, req *pb.AddOAuthProviderRequest) (*pb.AddOAuthProviderResponse, error) {
	return h.OAuthProviderHandler.AddOAuthProvider(ctx, req)
}

func (h *CombinedHandler) GetOAuthProvider(ctx context.Context, req *pb.GetOAuthProviderRequest) (*pb.GetOAuthProviderResponse, error) {
	return h.OAuthProviderHandler.GetOAuthProvider(ctx, req)
}

func (h *CombinedHandler) ListOAuthProviders(ctx context.Context, req *pb.ListOAuthProvidersRequest) (*pb.ListOAuthProvidersResponse, error) {
	return h.OAuthProviderHandler.ListOAuthProviders(ctx, req)
}

func (h *CombinedHandler) UpdateOAuthProvider(ctx context.Context, req *pb.UpdateOAuthProviderRequest) (*pb.UpdateOAuthProviderResponse, error) {
	return h.OAuthProviderHandler.UpdateOAuthProvider(ctx, req)
}

func (h *CombinedHandler) DeleteOAuthProvider(ctx context.Context, req *pb.DeleteOAuthProviderRequest) (*pb.DeleteOAuthProviderResponse, error) {
	return h.OAuthProviderHandler.DeleteOAuthProvider(ctx, req)
}
