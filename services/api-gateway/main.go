package main

import (
	"log"

	pb "github.com/ayushdevan01/AuthService/proto/developer"
	"github.com/ayushdevan01/AuthService/services/api-gateway/config"
	"github.com/ayushdevan01/AuthService/services/api-gateway/middleware"
	"github.com/ayushdevan01/AuthService/services/api-gateway/routes"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()

	devConn, err := grpc.NewClient(cfg.DeveloperServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Developer Service: %v", err)
	}
	defer devConn.Close()

	devClient := pb.NewDeveloperServiceClient(devConn)
	devRoutes := routes.NewDeveloperRoutes(devClient)

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

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

	log.Printf("API Gateway listening on port %s", cfg.HTTPPort)
	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
