package main

import (
	"log"

	pbDev "github.com/ayushdevan01/AuthService/proto/developer"
	pbToken "github.com/ayushdevan01/AuthService/proto/token"
	pbUser "github.com/ayushdevan01/AuthService/proto/user"
	"github.com/ayushdevan01/AuthService/services/api-gateway/config"
	"github.com/ayushdevan01/AuthService/services/api-gateway/middleware"
	"github.com/ayushdevan01/AuthService/services/api-gateway/routes"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()

	// Developer Service connection
	devConn, err := grpc.NewClient(cfg.DeveloperServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Developer Service: %v", err)
	}
	defer devConn.Close()

	// User Service connection
	userConn, err := grpc.NewClient(cfg.UserServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to User Service: %v", err)
	}
	defer userConn.Close()

	// Token Service connection
	tokenConn, err := grpc.NewClient(cfg.TokenServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Token Service: %v", err)
	}
	defer tokenConn.Close()

	// gRPC clients
	devClient := pbDev.NewDeveloperServiceClient(devConn)
	userClient := pbUser.NewUserServiceClient(userConn)
	tokenClient := pbToken.NewTokenServiceClient(tokenConn)

	// Route handlers
	devRoutes := routes.NewDeveloperRoutes(devClient, userClient)
	authRoutes := routes.NewAuthRoutes(userClient, tokenClient)

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// Developer routes (existing)
	dev := r.Group("/api/dev")
	{
		// Public routes
		dev.POST("/register", devRoutes.Register)
		dev.POST("/login", devRoutes.Login)

		// Protected routes
		auth := dev.Group("", middleware.AuthMiddleware(cfg.JWTSecret))
		{
			auth.POST("/logout", devRoutes.Logout)
			auth.GET("/profile", devRoutes.GetProfile)
			auth.PATCH("/profile", devRoutes.UpdateProfile)

			// App management
			auth.POST("/apps", devRoutes.CreateApp)
			auth.GET("/apps", devRoutes.ListApps)
			auth.GET("/apps/:id", devRoutes.GetApp)
			auth.PATCH("/apps/:id", devRoutes.UpdateApp)
			auth.DELETE("/apps/:id", devRoutes.DeleteApp)

			// App Users
			auth.GET("/apps/:id/users", devRoutes.ListUsers)

			// Key management
			auth.POST("/apps/:id/rotate-secret", devRoutes.RotateApiKey)
			auth.POST("/apps/:id/rotate-keys", devRoutes.RotateSigningKeys)
			auth.GET("/apps/:id/keys", devRoutes.ListSigningKeys)

			// OAuth provider config
			auth.POST("/apps/:id/providers", devRoutes.AddOAuthProvider)
			auth.GET("/apps/:id/providers", devRoutes.ListOAuthProviders)
			auth.PATCH("/apps/:id/providers/:provider", devRoutes.UpdateOAuthProvider)
			auth.DELETE("/apps/:id/providers/:provider", devRoutes.DeleteOAuthProvider)
		}
	}

	// OAuth & Auth routes (new - end-user facing)
	oauth := r.Group("/oauth")
	{
		oauth.GET("/authorize", authRoutes.Authorize)
		oauth.GET("/callback/:provider", authRoutes.Callback)
		oauth.POST("/refresh", authRoutes.RefreshTokens)
		oauth.POST("/revoke", authRoutes.RevokeToken)
	}

	// Email/Password auth routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", authRoutes.Register)
		auth.POST("/login", authRoutes.LoginWithEmail)
		auth.POST("/forgot-password", authRoutes.ForgotPassword)
		auth.POST("/reset-password", authRoutes.ResetPassword)
		auth.POST("/verify-email", authRoutes.VerifyEmail)
	}

	// Public API routes
	api := r.Group("/api/v1")
	{
		api.GET("/apps/:app_id/jwks", authRoutes.JWKS)
		api.POST("/verify", authRoutes.VerifyToken)
		api.GET("/userinfo", authRoutes.UserInfo)
	}

	log.Printf("API Gateway listening on port %s", cfg.HTTPPort)
	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
