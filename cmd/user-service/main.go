package main

import (
	"database/sql"
	"net"

	"github.com/dmehra2102/ecommerce-go-gRPC/internal/auth"
	"github.com/dmehra2102/ecommerce-go-gRPC/internal/config"
	"github.com/dmehra2102/ecommerce-go-gRPC/internal/database"
	"github.com/dmehra2102/ecommerce-go-gRPC/internal/logger"
	pb "github.com/dmehra2102/ecommerce-go-gRPC/proto/user"
	"github.com/dmehra2102/ecommerce-go-gRPC/services/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger.Init()
	log := logger.GetLogger()

	cfg := config.Load()

	db,err := database.NewPostgresConnection(&cfg.Database)
	if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

	// Create database tables
	if err := createTables(db); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiration)

	authInterceptor := auth.NewAuthInterceptor(jwtManager)

	repo := user.NewRepository(db)
	service := user.NewService(repo, jwtManager)

	// Create gRPC server with interceptors
	server := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.UnaryInterceptor),
		grpc.StreamInterceptor(authInterceptor.StreamInterceptor),
	)

	pb.RegisterUserServiceServer(server, service)

	 reflection.Register(server)

    // Start server
    lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    log.Infof("User service starting on port %s", cfg.Server.Port)
    if err := server.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}

func createTables(db *sql.DB) error {
	query := `
	CREATE TABEL IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(100) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
        first_name VARCHAR(100),
        last_name VARCHAR(100),
        role VARCHAR(50) DEFAULT 'user',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
    CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`

	_,err := db.Exec(query)
	return err
}