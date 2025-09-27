package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dmehra2102/ecommerce-go-gRPC/internal/auth"
	pb "github.com/dmehra2102/ecommerce-go-gRPC/proto/user"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	pb.UnimplementedUserServiceServer
	repo       *Repository
	jwtManager *auth.JWTManager
}

func NewService(repo *Repository, jwtManager *auth.JWTManager) *Service {
	return &Service{
		repo:       repo,
		jwtManager: jwtManager,
	}
}

func (s *Service) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Validate UserInput
	if req.Email == "" || req.Password == "" || req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password and username are required")
	}

	// Create user
	user, err := s.repo.CreateUser(req)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(user.Id, user.Email, user.Role)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb.RegisterResponse{
		User:  user,
		Token: token,
	}, nil
}

func (s *Service) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Validate UserInput
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	// Get user by email
	user, passwordHash, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "invalid credentials")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, err := s.jwtManager.GenerateToken(user.Id, user.Email, user.Role)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb.LoginResponse{
		User:  user,
		Token: token,
	}, nil
}

func (s *Service) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	user, err := s.repo.GetUserByID(req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}
	return &pb.GetUserResponse{User: user}, nil
}

func (s *Service) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	// Check if user is updating their own profile
	userID := ctx.Value("user_id").(string)
	if userID != req.Id {
		userRole := ctx.Value("user_role").(string)
		if userRole != "admin" {
			return nil, status.Error(codes.PermissionDenied, "can only update own profile")
		}
	}

	user, err := s.repo.UpdateUser(req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &pb.UpdateUserResponse{User: user}, nil
}

func (s *Service) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	// Check if user is admin
	userRole := ctx.Value("user_role").(string)
	if userRole != "admin" {
		return nil, status.Error(codes.PermissionDenied, "admin access required")
	}

	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	users, total, err := s.repo.ListUsers(req.Page, req.PageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list users")
	}

	return &pb.ListUsersResponse{
		Users: users,
		Total: total,
	}, nil
}
