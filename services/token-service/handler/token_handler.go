package handler

import (
	"context"
	"errors"

	pb "github.com/ayushdevan01/AuthService/proto/token"
	"github.com/ayushdevan01/AuthService/services/token-service/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TokenHandler struct {
	pb.UnimplementedTokenServiceServer
	tokenService *service.TokenService
}

func NewTokenHandler(tokenService *service.TokenService) *TokenHandler {
	return &TokenHandler{tokenService: tokenService}
}

func (h *TokenHandler) GenerateTokenPair(ctx context.Context, req *pb.GenerateTokenPairRequest) (*pb.GenerateTokenPairResponse, error) {
	result, err := h.tokenService.GenerateTokenPair(
		ctx, req.AppId, req.UserId, req.Email, req.Provider,
		req.EmailVerified, req.SessionId,
		req.AccessTokenTtl, req.RefreshTokenTtl,
	)
	if err != nil {
		if errors.Is(err, service.ErrNoSigningKey) {
			return nil, status.Errorf(codes.NotFound, "no signing key: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to generate tokens: %v", err)
	}

	return &pb.GenerateTokenPairResponse{
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	}, nil
}

func (h *TokenHandler) RefreshTokens(ctx context.Context, req *pb.RefreshTokensRequest) (*pb.RefreshTokensResponse, error) {
	accessToken, refreshToken, accessExp, refreshExp, err := h.tokenService.RefreshTokens(
		ctx, req.RefreshToken, req.AppId, req.RotateRefreshToken,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			return &pb.RefreshTokensResponse{
				Success:      false,
				ErrorCode:    "INVALID_REFRESH_TOKEN",
				ErrorMessage: "refresh token is invalid or expired",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to refresh tokens: %v", err)
	}

	resp := &pb.RefreshTokensResponse{
		Success:              true,
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessExp,
	}

	if refreshToken != nil {
		resp.RefreshToken = refreshToken
	}
	if refreshExp != nil {
		resp.RefreshTokenExpiresAt = refreshExp
	}

	return resp, nil
}

func (h *TokenHandler) RevokeToken(ctx context.Context, req *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	err := h.tokenService.RevokeToken(ctx, req.Token, req.TokenType, req.AppId)
	if err != nil {
		return &pb.RevokeTokenResponse{Success: false}, nil
	}
	return &pb.RevokeTokenResponse{Success: true}, nil
}

func (h *TokenHandler) GetPublicKeys(ctx context.Context, req *pb.GetPublicKeysRequest) (*pb.GetPublicKeysResponse, error) {
	keys, err := h.tokenService.GetPublicKeys(ctx, req.AppId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get public keys: %v", err)
	}

	pbKeys := make([]*pb.PublicKey, 0, len(keys))
	for _, k := range keys {
		pbKeys = append(pbKeys, &pb.PublicKey{
			Kid:       k.KID,
			PublicKey: k.PublicKey,
			IsActive:  k.IsActive,
		})
	}

	return &pb.GetPublicKeysResponse{Keys: pbKeys}, nil
}
