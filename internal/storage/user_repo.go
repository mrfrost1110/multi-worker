package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/multi-worker/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	db *Database
}

func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, req *model.RegisterRequest) (*model.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	var user model.User
	query := `
		INSERT INTO users (email, password, name, role, api_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, name, role, api_key, is_active, created_at, updated_at
	`
	err = r.db.QueryRowxContext(ctx, query, req.Email, string(hashedPassword), req.Name, model.UserRoleUser, apiKey).
		StructScan(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT id, email, password, name, role, api_key, is_active, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	query := `SELECT id, email, password, name, role, api_key, is_active, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByAPIKey(ctx context.Context, apiKey string) (*model.User, error) {
	var user model.User
	query := `SELECT id, email, password, name, role, api_key, is_active, created_at, updated_at FROM users WHERE api_key = $1 AND is_active = true`
	err := r.db.GetContext(ctx, &user, query, apiKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by API key: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) ValidatePassword(user *model.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	query := `UPDATE users SET updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	return err
}

func (r *UserRepository) RegenerateAPIKey(ctx context.Context, userID string) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", err
	}

	query := `UPDATE users SET api_key = $1, updated_at = $2 WHERE id = $3`
	_, err = r.db.ExecContext(ctx, query, apiKey, time.Now(), userID)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

func (r *UserRepository) CreateAdmin(ctx context.Context, email, password, name string) (*model.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	var user model.User
	query := `
		INSERT INTO users (email, password, name, role, api_key)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO UPDATE SET updated_at = CURRENT_TIMESTAMP
		RETURNING id, email, name, role, api_key, is_active, created_at, updated_at
	`
	err = r.db.QueryRowxContext(ctx, query, email, string(hashedPassword), name, model.UserRoleAdmin, apiKey).
		StructScan(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	return &user, nil
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
