package routes

import (
	"net/http"
	"strconv"

	pb "github.com/ayushdevan01/AuthService/proto/developer"
	pbUser "github.com/ayushdevan01/AuthService/proto/user"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeveloperRoutes struct {
	client     pb.DeveloperServiceClient
	userClient pbUser.UserServiceClient
}

func NewDeveloperRoutes(client pb.DeveloperServiceClient, userClient pbUser.UserServiceClient) *DeveloperRoutes {
	return &DeveloperRoutes{
		client:     client,
		userClient: userClient,
	}
}

func (dr *DeveloperRoutes) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := dr.client.Register(c.Request.Context(), &pb.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"developer":    formatDeveloper(resp.Developer),
		"access_token": resp.AccessToken,
	})
}

func (dr *DeveloperRoutes) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := dr.client.Login(c.Request.Context(), &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"developer":    formatDeveloper(resp.Developer),
		"access_token": resp.AccessToken,
	})
}

func (dr *DeveloperRoutes) Logout(c *gin.Context) {
	developerID := c.GetString("developer_id")

	_, err := dr.client.Logout(c.Request.Context(), &pb.LogoutRequest{
		DeveloperId: developerID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (dr *DeveloperRoutes) GetProfile(c *gin.Context) {
	developerID := c.GetString("developer_id")

	resp, err := dr.client.GetProfile(c.Request.Context(), &pb.GetProfileRequest{
		DeveloperId: developerID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.Found {
		c.JSON(http.StatusNotFound, gin.H{"error": "developer not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"developer": formatDeveloper(resp.Developer)})
}

func (dr *DeveloperRoutes) UpdateProfile(c *gin.Context) {
	developerID := c.GetString("developer_id")

	var req struct {
		Name *string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &pb.UpdateProfileRequest{DeveloperId: developerID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}

	resp, err := dr.client.UpdateProfile(c.Request.Context(), grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"developer": formatDeveloper(resp.Developer)})
}

// App Management

func (dr *DeveloperRoutes) CreateApp(c *gin.Context) {
	developerID := c.GetString("developer_id")

	var req struct {
		Name         string   `json:"name" binding:"required"`
		LogoURL      string   `json:"logo_url"`
		RedirectURLs []string `json:"redirect_urls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := dr.client.CreateApp(c.Request.Context(), &pb.CreateAppRequest{
		DeveloperId:  developerID,
		Name:         req.Name,
		LogoUrl:      req.LogoURL,
		RedirectUrls: req.RedirectURLs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"app":         formatApp(resp.App),
		"api_key":     resp.ApiKey,
		"signing_key": formatSigningKey(resp.SigningKey),
	})
}

func (dr *DeveloperRoutes) ListApps(c *gin.Context) {
	developerID := c.GetString("developer_id")

	resp, err := dr.client.ListApps(c.Request.Context(), &pb.ListAppsRequest{
		DeveloperId: developerID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	apps := make([]gin.H, 0, len(resp.Apps))
	for _, app := range resp.Apps {
		apps = append(apps, formatApp(app))
	}

	c.JSON(http.StatusOK, gin.H{"apps": apps})
}

func (dr *DeveloperRoutes) GetApp(c *gin.Context) {
	appID := c.Param("id")

	resp, err := dr.client.GetApp(c.Request.Context(), &pb.GetAppRequest{
		AppId: appID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.Found {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"app": formatApp(resp.App)})
}

func (dr *DeveloperRoutes) UpdateApp(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")

	var req struct {
		Name         *string  `json:"name"`
		LogoURL      *string  `json:"logo_url"`
		RedirectURLs []string `json:"redirect_urls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &pb.UpdateAppRequest{
		Id:           appID,
		DeveloperId:  developerID,
		RedirectUrls: req.RedirectURLs,
	}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.LogoURL != nil {
		grpcReq.LogoUrl = req.LogoURL
	}

	resp, err := dr.client.UpdateApp(c.Request.Context(), grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"app": formatApp(resp.App)})
}

func (dr *DeveloperRoutes) DeleteApp(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")

	_, err := dr.client.DeleteApp(c.Request.Context(), &pb.DeleteAppRequest{
		Id:          appID,
		DeveloperId: developerID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "app deleted"})
}

// Key Management

func (dr *DeveloperRoutes) RotateApiKey(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")

	resp, err := dr.client.RotateApiKey(c.Request.Context(), &pb.RotateApiKeyRequest{
		AppId:       appID,
		DeveloperId: developerID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"new_api_key": resp.NewApiKey})
}

func (dr *DeveloperRoutes) RotateSigningKeys(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")

	var req struct {
		GracePeriodHours int32 `json:"grace_period_hours"`
	}
	c.ShouldBindJSON(&req)

	resp, err := dr.client.RotateSigningKeys(c.Request.Context(), &pb.RotateSigningKeysRequest{
		AppId:            appID,
		DeveloperId:      developerID,
		GracePeriodHours: req.GracePeriodHours,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := gin.H{"new_key": formatSigningKey(resp.NewKey)}
	if resp.OldKey != nil {
		result["old_key"] = formatSigningKey(resp.OldKey)
	}

	c.JSON(http.StatusOK, result)
}

func (dr *DeveloperRoutes) ListSigningKeys(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")

	includeExpired := c.Query("include_expired") == "true"

	resp, err := dr.client.ListSigningKeys(c.Request.Context(), &pb.ListSigningKeysRequest{
		AppId:          appID,
		DeveloperId:    developerID,
		IncludeExpired: includeExpired,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	keys := make([]gin.H, 0, len(resp.Keys))
	for _, key := range resp.Keys {
		keys = append(keys, formatSigningKey(key))
	}

	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

// OAuth Provider Configuration

func (dr *DeveloperRoutes) AddOAuthProvider(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")

	var req struct {
		Provider     string   `json:"provider" binding:"required"`
		ClientID     string   `json:"client_id" binding:"required"`
		ClientSecret string   `json:"client_secret" binding:"required"`
		Scopes       []string `json:"scopes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := dr.client.AddOAuthProvider(c.Request.Context(), &pb.AddOAuthProviderRequest{
		AppId:        appID,
		DeveloperId:  developerID,
		Provider:     req.Provider,
		ClientId:     req.ClientID,
		ClientSecret: req.ClientSecret,
		Scopes:       req.Scopes,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"provider": formatOAuthProvider(resp.Provider)})
}

func (dr *DeveloperRoutes) ListOAuthProviders(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")

	resp, err := dr.client.ListOAuthProviders(c.Request.Context(), &pb.ListOAuthProvidersRequest{
		AppId:       appID,
		DeveloperId: developerID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	providers := make([]gin.H, 0, len(resp.Providers))
	for _, p := range resp.Providers {
		providers = append(providers, formatOAuthProvider(p))
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (dr *DeveloperRoutes) UpdateOAuthProvider(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")
	provider := c.Param("provider")

	var req struct {
		ClientID     *string  `json:"client_id"`
		ClientSecret *string  `json:"client_secret"`
		Scopes       []string `json:"scopes"`
		Enabled      *bool    `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := dr.client.UpdateOAuthProvider(c.Request.Context(), &pb.UpdateOAuthProviderRequest{
		AppId:        appID,
		DeveloperId:  developerID,
		Provider:     provider,
		ClientId:     req.ClientID,
		ClientSecret: req.ClientSecret,
		Scopes:       req.Scopes,
		Enabled:      req.Enabled,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"provider": formatOAuthProvider(resp.Provider)})
}

func (dr *DeveloperRoutes) DeleteOAuthProvider(c *gin.Context) {
	developerID := c.GetString("developer_id")
	appID := c.Param("id")
	provider := c.Param("provider")

	_, err := dr.client.DeleteOAuthProvider(c.Request.Context(), &pb.DeleteOAuthProviderRequest{
		AppId:       appID,
		DeveloperId: developerID,
		Provider:    provider,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider removed"})
}

// Helpers

func formatDeveloper(d *pb.Developer) gin.H {
	if d == nil {
		return nil
	}
	return gin.H{
		"id":         d.Id,
		"email":      d.Email,
		"name":       d.Name,
		"created_at": formatTimestamp(d.CreatedAt),
	}
}

func formatApp(a *pb.App) gin.H {
	if a == nil {
		return nil
	}
	return gin.H{
		"id":            a.Id,
		"app_id":        a.AppId,
		"developer_id":  a.DeveloperId,
		"name":          a.Name,
		"logo_url":      a.LogoUrl,
		"redirect_urls": a.RedirectUrls,
		"created_at":    formatTimestamp(a.CreatedAt),
		"updated_at":    formatTimestamp(a.UpdatedAt),
	}
}

func formatSigningKey(k *pb.SigningKey) gin.H {
	if k == nil {
		return nil
	}
	result := gin.H{
		"id":         k.Id,
		"app_id":     k.AppId,
		"kid":        k.Kid,
		"public_key": k.PublicKey,
		"is_active":  k.IsActive,
		"created_at": formatTimestamp(k.CreatedAt),
	}
	if k.ExpiresAt != nil {
		result["expires_at"] = formatTimestamp(k.ExpiresAt)
	}
	if k.RotatedAt != nil {
		result["rotated_at"] = formatTimestamp(k.RotatedAt)
	}
	return result
}

func formatOAuthProvider(p *pb.OAuthProvider) gin.H {
	if p == nil {
		return nil
	}
	return gin.H{
		"id":         p.Id,
		"app_id":     p.AppId,
		"provider":   p.Provider,
		"client_id":  p.ClientId,
		"scopes":     p.Scopes,
		"enabled":    p.Enabled,
		"created_at": formatTimestamp(p.CreatedAt),
	}
}

func formatTimestamp(ts *timestamppb.Timestamp) *string {
	if ts == nil {
		return nil
	}
	s := ts.AsTime().Format("2006-01-02T15:04:05Z")
	return &s
}

// GET /api/v1/apps/:id/users
func (dr *DeveloperRoutes) ListUsers(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app ID required"})
		return
	}

	pageSizeStr := c.Query("page_size")
	pageToken := c.Query("page_token")
	providerFilter := c.Query("provider_filter")
	emailSearch := c.Query("email_search")

	var pageSize int32 = 50
	if pageSizeStr != "" {
		if val, err := strconv.Atoi(pageSizeStr); err == nil && val > 0 {
			pageSize = int32(val)
		}
	}

	resp, err := dr.userClient.ListUsers(c.Request.Context(), &pbUser.ListUsersRequest{
		AppId:          appID,
		PageSize:       pageSize,
		PageToken:      pageToken,
		ProviderFilter: providerFilter,
		EmailSearch:    emailSearch,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
