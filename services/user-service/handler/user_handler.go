package handler

import (
	"context"
	"errors"
	"log"
	"time"

	pb "github.com/ayushdevan01/AuthService/proto/user"
	"github.com/ayushdevan01/AuthService/services/user-service/repository"
	"github.com/ayushdevan01/AuthService/services/user-service/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	userService          *service.UserService
	oauthService         *service.OAuthService
	sessionService       *service.SessionService
	passwordResetService *service.PasswordResetService
	emailService         *service.EmailService
	emailVerifService    *service.EmailVerificationService
}

func NewUserHandler(
	userService *service.UserService,
	oauthService *service.OAuthService,
	sessionService *service.SessionService,
	passwordResetService *service.PasswordResetService,
	emailService *service.EmailService,
	emailVerifService *service.EmailVerificationService,
) *UserHandler {
	return &UserHandler{
		userService:          userService,
		oauthService:         oauthService,
		sessionService:       sessionService,
		passwordResetService: passwordResetService,
		emailService:         emailService,
		emailVerifService:    emailVerifService,
	}
}

// --- User CRUD ---

func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	user, err := h.userService.CreateUser(
		ctx, req.AppId, req.Email,
		req.Name, req.AvatarUrl,
		req.Provider, req.ProviderUserId, req.PasswordHash,
		req.EmailVerified,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}
	return &pb.CreateUserResponse{User: toProtoUser(user)}, nil
}

func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := h.userService.GetUser(ctx, req.UserId, req.AppId)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return &pb.GetUserResponse{Found: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	return &pb.GetUserResponse{User: toProtoUser(user), Found: true}, nil
}

func (h *UserHandler) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.GetUserResponse, error) {
	user, err := h.userService.GetUserByEmail(ctx, req.AppId, req.Email)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return &pb.GetUserResponse{Found: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	return &pb.GetUserResponse{User: toProtoUser(user), Found: true}, nil
}

func (h *UserHandler) GetUserByProviderID(ctx context.Context, req *pb.GetUserByProviderIDRequest) (*pb.GetUserResponse, error) {
	user, err := h.userService.GetUserByProviderID(ctx, req.AppId, req.Provider, req.ProviderUserId)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return &pb.GetUserResponse{Found: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	return &pb.GetUserResponse{User: toProtoUser(user), Found: true}, nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	user, err := h.userService.UpdateUser(
		ctx, req.UserId, req.AppId,
		req.Email, req.PasswordHash, req.Name, req.AvatarUrl, req.EmailVerified,
	)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}
	return &pb.UpdateUserResponse{User: toProtoUser(user)}, nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	err := h.userService.DeleteUser(ctx, req.UserId, req.AppId)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return &pb.DeleteUserResponse{Success: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}
	return &pb.DeleteUserResponse{Success: true}, nil
}

func (h *UserHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	users, nextPageToken, totalCount, err := h.userService.ListUsers(
		ctx, req.AppId, int(req.PageSize), req.PageToken, req.ProviderFilter, req.EmailSearch,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	protoUsers := make([]*pb.User, 0, len(users))
	for _, u := range users {
		protoUsers = append(protoUsers, toProtoUser(u))
	}

	return &pb.ListUsersResponse{
		Users:         protoUsers,
		NextPageToken: nextPageToken,
		TotalCount:    int32(totalCount),
	}, nil
}

// --- OAuth ---

func (h *UserHandler) InitiateOAuth(ctx context.Context, req *pb.InitiateOAuthRequest) (*pb.InitiateOAuthResponse, error) {
	authURL, state, err := h.oauthService.InitiateOAuth(ctx, req.AppId, req.Provider, req.RedirectUri, req.CodeChallenge, req.CodeChallengeMethod, req.CodeVerifier)
	if err != nil {
		if errors.Is(err, service.ErrProviderNotConf) {
			return nil, status.Errorf(codes.NotFound, "OAuth provider not configured: %s", req.Provider)
		}
		return nil, status.Errorf(codes.Internal, "failed to initiate OAuth: %v", err)
	}
	return &pb.InitiateOAuthResponse{
		AuthorizationUrl: authURL,
		State:            state,
	}, nil
}

func (h *UserHandler) HandleOAuthCallback(ctx context.Context, req *pb.HandleOAuthCallbackRequest) (*pb.HandleOAuthCallbackResponse, error) {
	user, appID, redirectURI, isNewUser, err := h.oauthService.HandleOAuthCallback(ctx, req.Provider, req.Code, req.State)
	if err != nil {
		if errors.Is(err, service.ErrInvalidState) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid or expired OAuth state")
		}
		if errors.Is(err, service.ErrInvalidVerifier) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid PKCE code verifier")
		}
		return nil, status.Errorf(codes.Internal, "OAuth callback failed: %v", err)
	}
	return &pb.HandleOAuthCallbackResponse{
		User:        toProtoUser(user),
		AppId:       appID,
		RedirectUri: redirectURI,
		IsNewUser:   isNewUser,
	}, nil
}

// Email/Password Auth

func (h *UserHandler) RegisterWithEmail(ctx context.Context, req *pb.RegisterWithEmailRequest) (*pb.RegisterWithEmailResponse, error) {
	user, err := h.userService.RegisterWithEmail(ctx, req.AppId, req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			return nil, status.Errorf(codes.AlreadyExists, "user already exists with this email")
		}
		return nil, status.Errorf(codes.Internal, "registration failed: %v", err)
	}

	// Send verification email (non-blocking, uses secure random token)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.emailVerifService.SendVerification(ctx, req.AppId, user.ID, user.Email, user.Name); err != nil {
			log.Printf("Failed to send verification email to %s: %v", user.Email, err)
		}
	}()

	return &pb.RegisterWithEmailResponse{
		User:             toProtoUser(user),
		VerificationSent: true,
	}, nil
}

func (h *UserHandler) LoginWithEmail(ctx context.Context, req *pb.LoginWithEmailRequest) (*pb.LoginWithEmailResponse, error) {
	user, err := h.userService.LoginWithEmail(ctx, req.AppId, req.Email, req.Password, req.RequireEmailVerification)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAccountBlocked):
			return &pb.LoginWithEmailResponse{
				Success:      false,
				ErrorMessage: "account temporarily blocked - too many failed attempts",
			}, nil
		case errors.Is(err, service.ErrEmailNotVerified):
			return &pb.LoginWithEmailResponse{
				Success:              false,
				ErrorMessage:         "email not verified",
				RequiresVerification: true,
			}, nil
		case errors.Is(err, service.ErrInvalidPassword):
			return &pb.LoginWithEmailResponse{
				Success:      false,
				ErrorMessage: "invalid email or password",
			}, nil
		default:
			return nil, status.Errorf(codes.Internal, "login failed: %v", err)
		}
	}
	return &pb.LoginWithEmailResponse{
		User:    toProtoUser(user),
		Success: true,
	}, nil
}

func (h *UserHandler) ForgotPassword(ctx context.Context, req *pb.ForgotPasswordRequest) (*pb.ForgotPasswordResponse, error) {
	// Always returns success to avoid leaking whether the email exists
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := h.passwordResetService.InitiateReset(bgCtx, req.AppId, req.Email)
		if err != nil {
			log.Printf("ForgotPassword failed for email %s: %v", req.Email, err)
		}
	}()
	return &pb.ForgotPasswordResponse{EmailSent: true}, nil
}

func (h *UserHandler) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordResponse, error) {
	err := h.passwordResetService.ResetPassword(ctx, req.AppId, req.Token, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrResetTokenInvalid):
			return &pb.ResetPasswordResponse{Success: false, ErrorMessage: "invalid or expired reset token"}, nil
		case errors.Is(err, service.ErrResetTokenUsed):
			return &pb.ResetPasswordResponse{Success: false, ErrorMessage: "reset token already used"}, nil
		default:
			return nil, status.Errorf(codes.Internal, "password reset failed: %v", err)
		}
	}
	return &pb.ResetPasswordResponse{Success: true}, nil
}

func (h *UserHandler) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	userID, err := h.emailVerifService.VerifyEmail(ctx, req.AppId, req.Token)
	if err != nil {
		if errors.Is(err, service.ErrVerifyTokenInvalid) {
			return &pb.VerifyEmailResponse{Success: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "verification failed: %v", err)
	}
	return &pb.VerifyEmailResponse{Success: true, UserId: userID}, nil
}

// Session Management

func (h *UserHandler) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	result, err := h.sessionService.CreateSession(ctx, req.UserId, req.AppId, req.UserAgent, req.IpAddress, req.TtlSeconds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}
	return &pb.CreateSessionResponse{
		Session:      toProtoSession(result.Session),
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *UserHandler) GetSession(ctx context.Context, req *pb.GetSessionRequest) (*pb.GetSessionResponse, error) {
	session, found, expired, err := h.sessionService.GetSession(ctx, req.RefreshTokenHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get session: %v", err)
	}
	if !found {
		return &pb.GetSessionResponse{Found: false}, nil
	}
	return &pb.GetSessionResponse{
		Session: toProtoSession(session),
		Found:   true,
		Expired: expired,
	}, nil
}

func (h *UserHandler) RevokeSession(ctx context.Context, req *pb.RevokeSessionRequest) (*pb.RevokeSessionResponse, error) {
	err := h.sessionService.RevokeSession(ctx, req.UserId, req.SessionId)
	if err != nil {
		return &pb.RevokeSessionResponse{Success: false}, nil
	}
	return &pb.RevokeSessionResponse{Success: true}, nil
}

func (h *UserHandler) RevokeAllSessions(ctx context.Context, req *pb.RevokeAllSessionsRequest) (*pb.RevokeAllSessionsResponse, error) {
	count, err := h.sessionService.RevokeAllSessions(ctx, req.UserId, req.AppId, req.ExceptSessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to revoke sessions: %v", err)
	}
	return &pb.RevokeAllSessionsResponse{RevokedCount: int32(count)}, nil
}

func (h *UserHandler) ListSessions(ctx context.Context, req *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	sessions, err := h.sessionService.ListSessions(ctx, req.UserId, req.AppId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sessions: %v", err)
	}

	protoSessions := make([]*pb.Session, 0, len(sessions))
	for _, s := range sessions {
		protoSessions = append(protoSessions, toProtoSession(s))
	}

	return &pb.ListSessionsResponse{Sessions: protoSessions}, nil
}

// Activity Logging

func (h *UserHandler) LogActivity(ctx context.Context, req *pb.LogActivityRequest) (*pb.LogActivityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "activity logging is not implemented")
}

// --- Helpers ---

func toProtoUser(u *repository.User) *pb.User {
	if u == nil {
		return nil
	}
	protoUser := &pb.User{
		Id:             u.ID,
		AppId:          u.AppID,
		Email:          u.Email,
		Name:           u.Name,
		AvatarUrl:      u.AvatarURL,
		Provider:       u.Provider,
		ProviderUserId: u.ProviderUserID,
		EmailVerified:  u.EmailVerified,
		CreatedAt:      timestamppb.New(u.CreatedAt),
	}
	if u.LastLoginAt != nil {
		protoUser.LastLoginAt = timestamppb.New(*u.LastLoginAt)
	}
	return protoUser
}

func toProtoSession(s *repository.Session) *pb.Session {
	if s == nil {
		return nil
	}
	return &pb.Session{
		Id:        s.ID,
		UserId:    s.UserID,
		AppId:     s.AppID,
		UserAgent: s.UserAgent,
		IpAddress: s.IPAddress,
		CreatedAt: timestamppb.New(s.CreatedAt),
		ExpiresAt: timestamppb.New(s.ExpiresAt),
	}
}
