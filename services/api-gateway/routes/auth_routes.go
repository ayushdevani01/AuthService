package routes

import (
	"fmt"
	"net/http"
	"strings"

	pbToken "github.com/ayushdevan01/AuthService/proto/token"
	pbUser "github.com/ayushdevan01/AuthService/proto/user"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

// POST /auth/register
func (ar *AuthRoutes) Register(c *gin.Context) {
	var req struct {
		AppID    string `json:"app_id" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userResp, err := ar.userClient.RegisterWithEmail(c.Request.Context(), &pbUser.RegisterWithEmailRequest{
		AppId:    req.AppID,
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	sessionResp, err := ar.userClient.CreateSession(c.Request.Context(), &pbUser.CreateSessionRequest{
		UserId:     userResp.User.Id,
		AppId:      req.AppID,
		UserAgent:  c.Request.UserAgent(),
		IpAddress:  c.ClientIP(),
		TtlSeconds: 30 * 24 * 60 * 60,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	tokenResp, err := ar.tokenClient.GenerateTokenPair(c.Request.Context(), &pbToken.GenerateTokenPairRequest{
		AppId:         req.AppID,
		UserId:        userResp.User.Id,
		Email:         userResp.User.Email,
		Provider:      "email",
		EmailVerified: userResp.User.EmailVerified,
		SessionId:     sessionResp.Session.Id,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"access_token":      tokenResp.AccessToken,
		"refresh_token":     sessionResp.RefreshToken,
		"token_type":        "Bearer",
		"expires_in":        tokenResp.AccessTokenExpiresAt,
		"verification_sent": userResp.VerificationSent,
	})
}

// POST /auth/login
func (ar *AuthRoutes) LoginWithEmail(c *gin.Context) {
	var req struct {
		AppID    string `json:"app_id" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loginResp, err := ar.userClient.LoginWithEmail(c.Request.Context(), &pbUser.LoginWithEmailRequest{
		AppId:    req.AppID,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !loginResp.Success {
		c.JSON(http.StatusUnauthorized, gin.H{"error": loginResp.ErrorMessage})
		return
	}

	sessionResp, err := ar.userClient.CreateSession(c.Request.Context(), &pbUser.CreateSessionRequest{
		UserId:     loginResp.User.Id,
		AppId:      req.AppID,
		UserAgent:  c.Request.UserAgent(),
		IpAddress:  c.ClientIP(),
		TtlSeconds: 30 * 24 * 60 * 60,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	tokenResp, err := ar.tokenClient.GenerateTokenPair(c.Request.Context(), &pbToken.GenerateTokenPairRequest{
		AppId:         req.AppID,
		UserId:        loginResp.User.Id,
		Email:         loginResp.User.Email,
		Provider:      "email",
		EmailVerified: loginResp.User.EmailVerified,
		SessionId:     sessionResp.Session.Id,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": sessionResp.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    tokenResp.AccessTokenExpiresAt,
	})
}

// POST /auth/forgot-password
func (ar *AuthRoutes) ForgotPassword(c *gin.Context) {
	var req struct {
		AppID string `json:"app_id" binding:"required"`
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ar.userClient.ForgotPassword(c.Request.Context(), &pbUser.ForgotPasswordRequest{
		AppId: req.AppID,
		Email: req.Email,
	})

	// Always return success regardless of whether email exists
	c.JSON(http.StatusOK, gin.H{"message": "if an account with that email exists, a reset link has been sent"})
}

// POST /auth/reset-password
func (ar *AuthRoutes) ResetPassword(c *gin.Context) {
	var req struct {
		AppID       string `json:"app_id" binding:"required"`
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ar.userClient.ResetPassword(c.Request.Context(), &pbUser.ResetPasswordRequest{
		AppId:       req.AppID,
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": resp.ErrorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// POST /auth/verify-email
func (ar *AuthRoutes) VerifyEmail(c *gin.Context) {
	var req struct {
		AppID string `json:"app_id" binding:"required"`
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ar.userClient.VerifyEmail(c.Request.Context(), &pbUser.VerifyEmailRequest{
		AppId: req.AppID,
		Token: req.Token,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully", "user_id": resp.UserId})
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// POST /api/v1/verify
func (ar *AuthRoutes) VerifyToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
		AppID string `json:"app_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Fetch public keys from token service
		resp, err := ar.tokenClient.GetPublicKeys(c.Request.Context(), &pbToken.GetPublicKeysRequest{
			AppId: req.AppID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch public keys: %w", err)
		}

		// Find the matching key
		for _, k := range resp.Keys {
			if k.Kid == kid {
				// Parse the PEM public key
				pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(k.PublicKey))
				if err != nil {
					return nil, fmt.Errorf("failed to parse public key: %w", err)
				}
				return pubKey, nil
			}
		}

		return nil, fmt.Errorf("public key '%s' not found for app '%s'", kid, req.AppID)
	})

	if err != nil || !token.Valid {
		errMsg := "invalid token"
		if err != nil {
			errMsg = err.Error()
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": errMsg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"claims": token.Claims,
	})
}

// GET /api/v1/userinfo
func (ar *AuthRoutes) UserInfo(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, use: Bearer <token>"})
		return
	}

	tokenString := parts[1]
	
	// Temporarily parse without fully verifying just to extract `aud` (App ID)
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	appID, ok := claims["aud"].(string)
	if !ok || appID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing aud (app_id) claim"})
		return
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing sub (user_id) claim"})
		return
	}

	// Now properly verify the token using the extracted AppID to fetch public keys
	verifiedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}

		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid")
		}

		resp, err := ar.tokenClient.GetPublicKeys(c.Request.Context(), &pbToken.GetPublicKeysRequest{
			AppId: appID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch public keys")
		}

		for _, k := range resp.Keys {
			if k.Kid == kid {
				return jwt.ParseRSAPublicKeyFromPEM([]byte(k.PublicKey))
			}
		}
		return nil, fmt.Errorf("public key not found")
	})

	if err != nil || !verifiedToken.Valid {
		errMsg := "invalid or expired token"
		if err != nil {
			errMsg = err.Error()
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": errMsg})
		return
	}

	// Token is fully verified. Call userClient to get full profile.
	userResp, err := ar.userClient.GetUser(c.Request.Context(), &pbUser.GetUserRequest{
		AppId:  appID,
		UserId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user profile: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, userResp.User)
}
