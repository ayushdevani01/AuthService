package main

import (
	"log"
	"net"

	pb "github.com/ayushdevan01/AuthService/proto/user"
	"github.com/ayushdevan01/AuthService/services/user-service/config"
	"github.com/ayushdevan01/AuthService/services/user-service/database"
	"github.com/ayushdevan01/AuthService/services/user-service/handler"
	"github.com/ayushdevan01/AuthService/services/user-service/repository"
	"github.com/ayushdevan01/AuthService/services/user-service/service"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	db := database.ConnectPostgres(cfg.DatabaseURL)
	defer db.Close()

	redisClient := database.ConnectRedis(cfg.RedisURL)
	defer redisClient.Close()

	// Repositories
	userRepo := repository.NewUserRepository(db)
	identityRepo := repository.NewIdentityRepository(db)
	sessionRepo := repository.NewSessionRepository(db, redisClient)
	resetRepo := repository.NewPasswordResetRepository(db)

	// Services
	rateLimiter := service.NewLoginRateLimiter(redisClient)
	emailSvc := service.NewEmailService(cfg.ResendAPIKey, cfg.PlatformURL, cfg.EmailFrom)
	userSvc := service.NewUserService(userRepo, identityRepo, rateLimiter)
	oauthSvc := service.NewOAuthService(redisClient, db, userSvc, cfg.EncryptionKey, cfg.APIPublicURL)
	sessionSvc := service.NewSessionService(sessionRepo)
	resetSvc := service.NewPasswordResetService(resetRepo, identityRepo, userRepo, emailSvc, redisClient)
	emailVerifSvc := service.NewEmailVerificationService(userRepo, emailSvc, redisClient)

	h := handler.NewUserHandler(userSvc, oauthSvc, sessionSvc, resetSvc, emailSvc, emailVerifSvc)

	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, h)

	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	log.Printf("User Service listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
