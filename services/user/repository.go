package user

import (
	"database/sql"
	"time"

	pb "github.com/dmehra2102/ecommerce-go-gRPC/proto/user"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CreateUser(req *pb.RegisterRequest) (*pb.User, error) {
	id := uuid.New().String()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO users (id, email, username, password_hash, first_name, last_name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, email, username, first_name, last_name, role, created_at, updated_at
	`

	now := time.Now()
	var user pb.User
	var createdAt, updatedAt time.Time

	err = r.db.QueryRow(query, id, req.Email, req.Username, string(hashedPassword), req.FirstName, req.LastName, "user", now, now).Scan(&user.Id, &user.Email, &user.Username, &user.FirstName, &user.LastName, &user.Role, &createdAt, &updatedAt)

	if err != nil {
		return nil, err
	}

	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &user, nil
}

func (r *Repository) GetUserByEmail(email string) (*pb.User, string, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, role, created_at, updated_at
		FROM users WHERE email = $1`

	var user pb.User
	var passwordHash string
	var createdAt, updatedAt time.Time

	r.db.QueryRow(query, email).Scan(&user.Id, &user.Email, &user.Username, &passwordHash,
		&user.FirstName, &user.LastName, &user.Role, &createdAt, &updatedAt)

	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &user, passwordHash, nil
}

func (r *Repository) GetUserByID(id string) (*pb.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, role, created_at, updated_at
		FROM users WHERE id = $1`

	var user pb.User
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(query, id).Scan(
		user.Id, &user.Email, &user.Username,
		&user.FirstName, &user.LastName, &user.Role, &createdAt, &updatedAt)

	if err != nil {
		return nil, err
	}

	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &user, nil
}

func (r *Repository) UpdateUser(req *pb.UpdateUserRequest) (*pb.User, error) {
	query := `
        UPDATE users SET first_name = $1, last_name = $2, username = $3, updated_at = $4
        WHERE id = $5
        RETURNING id, email, username, first_name, last_name, role, created_at, updated_at`

	var user pb.User
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(query, req.FirstName, req.LastName, req.Username, time.Now(), req.Id).Scan(
		&user.Id, &user.Email, &user.Username, &user.FirstName,
		&user.LastName, &user.Role, &createdAt, &updatedAt)

	if err != nil {
		return nil, err
	}

	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &user, nil
}

func (r *Repository) ListUsers(page, pageSize int32) ([]*pb.User, int32, error) {
	offset := (page - 1) * pageSize

	query := `
		SELECT id, email, username, first_name, last_name, role, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*pb.User
	for rows.Next() {
		var user pb.User
		var createdAt, updatedAt time.Time

		err := rows.Scan(&user.Id, &user.Email, &user.Username,
			&user.FirstName, &user.LastName, &user.Role, &createdAt, &updatedAt)
		if err != nil {
			return nil, 0, err
		}

		user.CreatedAt = timestamppb.New(createdAt)
		user.UpdatedAt = timestamppb.New(updatedAt)
		users = append(users, &user)
	}

	// Get total count
	var total int32
	countQuery := "SELECT COUNT(*) FROM users"
	err = r.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
