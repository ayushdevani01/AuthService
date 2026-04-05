package routes

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	pbDev "github.com/ayushdevan01/AuthService/proto/developer"
	pbToken "github.com/ayushdevan01/AuthService/proto/token"
	pbUser "github.com/ayushdevan01/AuthService/proto/user"
	"github.com/ayushdevan01/AuthService/services/api-gateway/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthRoutes struct {
	devClient   pbDev.DeveloperServiceClient
	userClient  pbUser.UserServiceClient
	tokenClient pbToken.TokenServiceClient
	appResolver *middleware.AppResolver
}

func NewAuthRoutes(devClient pbDev.DeveloperServiceClient, userClient pbUser.UserServiceClient, tokenClient pbToken.TokenServiceClient, appResolver *middleware.AppResolver) *AuthRoutes {
	return &AuthRoutes{
		devClient:   devClient,
		userClient:  userClient,
		tokenClient: tokenClient,
		appResolver: appResolver,
	}
}

func buildRedirectURLWithFragment(redirectURI string, values url.Values) string {
	fragment := values.Encode()
	if fragment == "" {
		return redirectURI
	}
	return redirectURI + "#" + fragment
}

func isAllowedRedirect(app *pbDev.App, redirectURI string) bool {
	if app == nil || redirectURI == "" {
		return false
	}
	for _, allowed := range app.RedirectUrls {
		if allowed == redirectURI {
			return true
		}
	}
	return false
}

func (ar *AuthRoutes) getPublicApp(c *gin.Context, publicAppID string) (*pbDev.App, error) {
	resp, err := ar.devClient.GetPublicApp(c.Request.Context(), &pbDev.GetPublicAppRequest{AppId: publicAppID})
	if err != nil {
		return nil, err
	}
	if !resp.Found || resp.App == nil {
		return nil, fmt.Errorf("app not found")
	}
	return resp.App, nil
}

// GET /oauth/authorize?app_id=xxx&provider=google&redirect_uri=...&code_challenge=...&code_challenge_method=S256
func (ar *AuthRoutes) Authorize(c *gin.Context) {
	appID := c.Query("app_id")
	provider := c.Query("provider")
	redirectURI := c.Query("redirect_uri")
	codeChallenge := c.Query("code_challenge")
	challengeMethod := c.Query("code_challenge_method")
	codeVerifier := c.Query("code_verifier")

	if appID == "" || provider == "" || redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id, provider, and redirect_uri are required"})
		return
	}

	// TODO : add chaching for this
	// appID -> WHOLE APP
	// invalidate -> when app is updated or deleted
	publicApp, err := ar.getPublicApp(c, appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}
	if !isAllowedRedirect(publicApp, redirectURI) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid redirect_uri"})
		return
	}

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	resp, err := ar.userClient.InitiateOAuth(c.Request.Context(), &pbUser.InitiateOAuthRequest{
		AppId:               resolvedAppID,
		Provider:            provider,
		RedirectUri:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: challengeMethod,
		CodeVerifier:        codeVerifier,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate OAuth"})
		return
	}

	// Redirect user to the OAuth provider
	c.Redirect(http.StatusFound, resp.AuthorizationUrl)
}

// GET /oauth/callback/:provider?code=xxx&state=xxx&code_verifier=...
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth callback failed"})
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
		Provider:      provider,
		EmailVerified: callbackResp.User.EmailVerified,
		SessionId:     sessionResp.Session.Id,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// Redirect back to the developer's app with tokens
	redirectURI := callbackResp.RedirectUri

	refreshToken := tokenResp.RefreshToken
	if refreshToken == "" {
		refreshToken = sessionResp.RefreshToken
	}

	redirectValues := url.Values{}
	redirectValues.Set("access_token", tokenResp.AccessToken)
	redirectValues.Set("refresh_token", refreshToken)
	redirectValues.Set("token_type", "Bearer")
	redirectValues.Set("expires_at", fmt.Sprintf("%d", tokenResp.AccessTokenExpiresAt))
	redirectURL := buildRedirectURLWithFragment(redirectURI, redirectValues)

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

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	resp, err := ar.tokenClient.RefreshTokens(c.Request.Context(), &pbToken.RefreshTokensRequest{
		RefreshToken:       req.RefreshToken,
		AppId:              resolvedAppID,
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
		"expires_at":   resp.AccessTokenExpiresAt,
	}

	if resp.RefreshToken != nil {
		result["refresh_token"] = *resp.RefreshToken
	}
	if resp.RefreshTokenExpiresAt != nil {
		result["refresh_token_expires_at"] = *resp.RefreshTokenExpiresAt
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

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	resp, err := ar.tokenClient.RevokeToken(c.Request.Context(), &pbToken.RevokeTokenRequest{
		Token:     req.Token,
		TokenType: req.TokenType,
		AppId:     resolvedAppID,
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
	resolvedAppID, err := ar.appResolver.ResolveAppID(c, appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	// TODO: Add caching for public keys
	// app_id -> keys
	resp, err := ar.tokenClient.GetPublicKeys(c.Request.Context(), &pbToken.GetPublicKeysRequest{
		AppId: resolvedAppID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	keys := make([]gin.H, 0, len(resp.Keys))
	for _, k := range resp.Keys {
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(k.PublicKey))
		if err != nil {
			continue
		}

		keys = append(keys, gin.H{
			"kid": k.Kid,
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pubKey.E)).Bytes()),
		})
	}

	if len(keys) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load valid public keys"})
		return
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

	publicApp, err := ar.getPublicApp(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	userResp, err := ar.userClient.RegisterWithEmail(c.Request.Context(), &pbUser.RegisterWithEmailRequest{
		AppId:    resolvedAppID,
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	if publicApp.RequireEmailVerification {
		c.JSON(http.StatusCreated, gin.H{
			"verification_sent":     userResp.VerificationSent,
			"requires_verification": true,
			"message":               "account created, please verify your email before signing in",
		})
		return
	}

	sessionResp, err := ar.userClient.CreateSession(c.Request.Context(), &pbUser.CreateSessionRequest{
		UserId:     userResp.User.Id,
		AppId:      resolvedAppID,
		UserAgent:  c.Request.UserAgent(),
		IpAddress:  c.ClientIP(),
		TtlSeconds: 30 * 24 * 60 * 60,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	tokenResp, err := ar.tokenClient.GenerateTokenPair(c.Request.Context(), &pbToken.GenerateTokenPairRequest{
		AppId:         resolvedAppID,
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

	refreshToken := tokenResp.RefreshToken
	if refreshToken == "" {
		refreshToken = sessionResp.RefreshToken
	}

	c.JSON(http.StatusCreated, gin.H{
		"access_token":             tokenResp.AccessToken,
		"refresh_token":            refreshToken,
		"token_type":               "Bearer",
		"expires_at":               tokenResp.AccessTokenExpiresAt,
		"refresh_token_expires_at": tokenResp.RefreshTokenExpiresAt,
		"verification_sent":        userResp.VerificationSent,
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

	publicApp, err := ar.getPublicApp(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	loginResp, err := ar.userClient.LoginWithEmail(c.Request.Context(), &pbUser.LoginWithEmailRequest{
		AppId:                    resolvedAppID,
		Email:                    req.Email,
		Password:                 req.Password,
		RequireEmailVerification: publicApp.RequireEmailVerification,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !loginResp.Success {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":                 loginResp.ErrorMessage,
			"requires_verification": loginResp.RequiresVerification,
		})
		return
	}

	sessionResp, err := ar.userClient.CreateSession(c.Request.Context(), &pbUser.CreateSessionRequest{
		UserId:     loginResp.User.Id,
		AppId:      resolvedAppID,
		UserAgent:  c.Request.UserAgent(),
		IpAddress:  c.ClientIP(),
		TtlSeconds: 30 * 24 * 60 * 60,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	tokenResp, err := ar.tokenClient.GenerateTokenPair(c.Request.Context(), &pbToken.GenerateTokenPairRequest{
		AppId:         resolvedAppID,
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

	refreshToken := tokenResp.RefreshToken
	if refreshToken == "" {
		refreshToken = sessionResp.RefreshToken
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":             tokenResp.AccessToken,
		"refresh_token":            refreshToken,
		"token_type":               "Bearer",
		"expires_at":               tokenResp.AccessTokenExpiresAt,
		"refresh_token_expires_at": tokenResp.RefreshTokenExpiresAt,
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

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	ar.userClient.ForgotPassword(c.Request.Context(), &pbUser.ForgotPasswordRequest{
		AppId: resolvedAppID,
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

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	resp, err := ar.userClient.ResetPassword(c.Request.Context(), &pbUser.ResetPasswordRequest{
		AppId:       resolvedAppID,
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

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, req.AppID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	resp, err := ar.userClient.VerifyEmail(c.Request.Context(), &pbUser.VerifyEmailRequest{
		AppId: resolvedAppID,
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

		resolvedAppID, err := ar.appResolver.ResolveAppID(c, req.AppID)
		if err != nil {
			return nil, fmt.Errorf("invalid app_id")
		}
		// Fetch public keys from token service
		resp, err := ar.tokenClient.GetPublicKeys(c.Request.Context(), &pbToken.GetPublicKeysRequest{
			AppId: resolvedAppID,
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

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
		return
	}

	kid := ""
	if kidVal, ok := token.Header["kid"].(string); ok {
		kid = kidVal
	}
	if kid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing kid in token header"})
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

	resolvedAppID, err := ar.appResolver.ResolveAppID(c, appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	resp, err := ar.tokenClient.GetPublicKeys(c.Request.Context(), &pbToken.GetPublicKeysRequest{
		AppId: resolvedAppID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch public keys"})
		return
	}

	var pubKey interface{}
	for _, k := range resp.Keys {
		if k.Kid == kid {
			pubKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(k.PublicKey))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse public key"})
				return
			}
			break
		}
	}
	if pubKey == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "public key not found"})
		return
	}

	// Verify token
	verifiedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return pubKey, nil
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

	c.JSON(http.StatusOK, gin.H{"user": formatUser(userResp.User)})
}
