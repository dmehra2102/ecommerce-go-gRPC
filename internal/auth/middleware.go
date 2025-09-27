package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	jwtManager    *JWTManager
	publicMehtods map[string]bool
}

func NewAuthInterceptor(jwtManager *JWTManager) *AuthInterceptor {
	publicMethods := map[string]bool{
		"/user.UserService/Register":           true,
		"/user.UserService/Login":              true,
		"/product.ProductService/ListProducts": true,
		"/product.ProductService/GetProduct":   true,
	}

	return &AuthInterceptor{
		jwtManager:    jwtManager,
		publicMehtods: publicMethods,
	}
}

type contextKey string

const (
	userIDKey    contextKey = "user_id"
	userEmailKey contextKey = "user_email"
	userRoleKey  contextKey = "user_role"
)

func (interceptor *AuthInterceptor) UnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// Skip authentication for public methods
	if interceptor.publicMehtods[info.FullMethod] {
		return handler(ctx, req)
	}

	claims, err := interceptor.authorize(ctx)
	if err != nil {
		return nil, err
	}

	// Adding Claims to context using custom context keys
	ctx = context.WithValue(ctx, userIDKey, claims.UserID)
	ctx = context.WithValue(ctx, userEmailKey, claims.Email)
	ctx = context.WithValue(ctx, userRoleKey, claims.Role)

	return handler(ctx, req)
}

func (interceptor *AuthInterceptor) StreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// Skip authentication for public methods
	if interceptor.publicMehtods[info.FullMethod] {
		return handler(srv, ss)
	}

	claims, err := interceptor.authorize(ss.Context())
	if err != nil {
		return err
	}

	// Create new context with claims
	ctx := context.WithValue(ss.Context(), userIDKey, claims.UserID)
	ctx = context.WithValue(ctx, userEmailKey, claims.Email)
	ctx = context.WithValue(ctx, userRoleKey, claims.Role)

	wrapped := &wrappedStream{
		ServerStream: ss,
		ctx:          ctx,
	}

	return handler(srv, wrapped)
}

func (interceptor *AuthInterceptor) authorize(ctx context.Context) (*Claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata not provided")
	}

	authHeader := md["authorization"]
	if len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authoriation token not provided")
	}

	token := authHeader[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	token = strings.TrimPrefix(token, "Bearer ")
	claims, err := interceptor.jwtManager.ValidateToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return claims, nil
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
