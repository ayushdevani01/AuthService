package main

import (
	"log"
	"net"

	pb "github.com/ayushdevan01/AuthService/proto/token"
	"github.com/ayushdevan01/AuthService/services/token-service/config"
	"github.com/ayushdevan01/AuthService/services/token-service/database"
	"github.com/ayushdevan01/AuthService/services/token-service/handler"
	"github.com/ayushdevan01/AuthService/services/token-service/repository"
	"github.com/ayushdevan01/AuthService/services/token-service/service"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	db := database.ConnectPostgres(cfg.DatabaseURL)
	defer db.Close()

	redisClient := database.ConnectRedis(cfg.RedisURL)
	defer redisClient.Close()

	// Repositories
	signingKeyRepo := repository.NewSigningKeyRepository(db, cfg.EncryptionKey)
	sessionRepo := repository.NewSessionRepository(db, redisClient)

	// Service
	tokenSvc := service.NewTokenService(signingKeyRepo, sessionRepo)

	// Handler
	h := handler.NewTokenHandler(tokenSvc)

	grpcServer := grpc.NewServer()
	pb.RegisterTokenServiceServer(grpcServer, h)

	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	log.Printf("Token Service listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
