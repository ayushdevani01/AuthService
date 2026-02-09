package main

import (
	"log"
	"net"

	pb "github.com/ayushdevan01/AuthService/proto/developer"
	"github.com/ayushdevan01/AuthService/services/developer-service/config"
	"github.com/ayushdevan01/AuthService/services/developer-service/database"
	"github.com/ayushdevan01/AuthService/services/developer-service/handler"
	"github.com/ayushdevan01/AuthService/services/developer-service/repository"
	"github.com/ayushdevan01/AuthService/services/developer-service/service"
	"google.golang.org/grpc"
)

func main() {

	cfg := config.Load()

	db := database.Connect(cfg.DatabaseURL)
	defer db.Close()

	repo := repository.NewDeveloperRepository(db)
	svc := service.NewDeveloperService(repo, cfg.JWTSecret)
	h := handler.NewDeveloperHandler(svc)

	grpcServer := grpc.NewServer()
	pb.RegisterDeveloperServiceServer(grpcServer, h)

	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	log.Printf("Developer Service listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
