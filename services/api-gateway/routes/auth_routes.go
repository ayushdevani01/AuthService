package routes

import (
	"fmt"
	"net/http"

	pbToken "github.com/ayushdevan01/AuthService/proto/token"
	pbUser "github.com/ayushdevan01/AuthService/proto/user"
	"github.com/gin-gonic/gin"
)

type AuthRoutes struct {
	userClient  pbUser.UserServiceClient
	tokenClient pbToken.TokenServiceClient
}

func NewAuthRoutes(userClient pbUser.UserServiceClient, tokenClient pbToken.TokenServiceClient) *AuthRoutes {
	return &AuthRoutes{
		userClient:  userClient,
		tokenClient: tokenClient,
	}
}

// GET /oauth/authorize?app_id=xxx&provider=google&redirect_uri=...
func (ar *AuthRoutes) Authorize(c *gin.Context) {
	appID := c.Query("app_id")
	provider := c.Query("provider")
	redirectURI := c.Query("redirect_uri")

	if appID == "" || provider == "" || redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id, provider, and redirect_uri are required"})
		return
	}

	resp, err := ar.userClient.InitiateOAuth(c.Request.Context(), &pbUser.InitiateOAuthRequest{
		AppId:       appID,
		Provider:    provider,
		RedirectUri: redirectURI,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Redirect user to the OAuth provider
	c.Redirect(http.StatusFound, resp.AuthorizationUrl)
}

// GET /oauth/callback/:provider?code=xxx&state=xxx
func (ar *AuthRoutes) Callback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		// Check for error from provider
		oauthErr := c.Query("error")
		if oauthErr != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("OAuth error: %s", oauthErr)})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and state are required"})
		return
	}

	// Handle OAuth callback → get user
	callbackResp, err := ar.userClient.HandleOAuthCallback(c.Request.Context(), &pbUser.HandleOAuthCallbackRequest{
		Provider: provider,
		Code:     code,
		State:    state,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create session
	sessionResp, err := ar.userClient.CreateSession(c.Request.Context(), &pbUser.CreateSessionRequest{
		UserId:     callbackResp.User.Id,
		AppId:      callbackResp.AppId,
		UserAgent:  c.Request.UserAgent(),
		IpAddress:  c.ClientIP(),
		TtlSeconds: 30 * 24 * 60 * 60, // 30 days
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Generate token pair
	tokenResp, err := ar.tokenClient.GenerateTokenPair(c.Request.Context(), &pbToken.GenerateTokenPairRequest{
		AppId:         callbackResp.AppId,
		UserId:        callbackResp.User.Id,
		Email:         callbackResp.User.Email,
		Provider:      callbackResp.User.Provider,
		EmailVerified: callbackResp.User.EmailVerified,
		SessionId:     sessionResp.Session.Id,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// Redirect back to the developer's app with tokens
	redirectURI := callbackResp.RedirectUri
	separator := "?"
	if contains(redirectURI, "?") {
		separator = "&"
	}

	redirectURL := fmt.Sprintf("%s%saccess_token=%s&refresh_token=%s&token_type=Bearer&expires_in=%d",
		redirectURI, separator,
		tokenResp.AccessToken,
		sessionResp.RefreshToken,
		tokenResp.AccessTokenExpiresAt,
	)

	c.Redirect(http.StatusFound, redirectURL)
}

// POST /oauth/refresh
func (ar *AuthRoutes) RefreshTokens(c *gin.Context) {
	var req struct {
		RefreshToken       string `json:"refresh_token" binding:"required"`
		AppID              string `json:"app_id" binding:"required"`
		RotateRefreshToken bool   `json:"rotate_refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ar.tokenClient.RefreshTokens(c.Request.Context(), &pbToken.RefreshTokensRequest{
		RefreshToken:       req.RefreshToken,
		AppId:              req.AppID,
		RotateRefreshToken: req.RotateRefreshToken,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      resp.ErrorCode,
			"error_desc": resp.ErrorMessage,
		})
		return
	}

	result := gin.H{
		"access_token": resp.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   resp.AccessTokenExpiresAt,
	}

	if resp.RefreshToken != nil {
		result["refresh_token"] = *resp.RefreshToken
	}
	if resp.RefreshTokenExpiresAt != nil {
		result["refresh_token_expires_in"] = *resp.RefreshTokenExpiresAt
	}

	c.JSON(http.StatusOK, result)
}

// POST /oauth/revoke
func (ar *AuthRoutes) RevokeToken(c *gin.Context) {
	var req struct {
		Token     string `json:"token" binding:"required"`
		TokenType string `json:"token_type" binding:"required"` // "access" or "refresh"
		AppID     string `json:"app_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ar.tokenClient.RevokeToken(c.Request.Context(), &pbToken.RevokeTokenRequest{
		Token:     req.Token,
		TokenType: req.TokenType,
		AppId:     req.AppID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": resp.Success})
}

// GET /api/v1/apps/:app_id/jwks
func (ar *AuthRoutes) JWKS(c *gin.Context) {
	appID := c.Param("app_id")

	resp, err := ar.tokenClient.GetPublicKeys(c.Request.Context(), &pbToken.GetPublicKeysRequest{
		AppId: appID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	keys := make([]gin.H, 0, len(resp.Keys))
	for _, k := range resp.Keys {
		keys = append(keys, gin.H{
			"kid":       k.Kid,
			"kty":       "RSA",
			"alg":       "RS256",
			"use":       "sig",
			"is_active": k.IsActive,
			"key":       k.PublicKey,
		})
	}

	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
