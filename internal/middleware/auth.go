package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/multi-worker/internal/config"
	"github.com/multi-worker/internal/model"
	"github.com/multi-worker/internal/storage"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware handles JWT and API key authentication
type AuthMiddleware struct {
	jwtSecret []byte
	userRepo  *storage.UserRepository
	expHours  int
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(cfg config.JWTConfig, userRepo *storage.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(cfg.Secret),
		userRepo:  userRepo,
		expHours:  cfg.ExpirationHours,
	}
}

// Authenticate middleware checks for valid JWT or API key
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try JWT token first
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := m.ValidateToken(tokenStr)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try API key
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			user, err := m.userRepo.FindByAPIKey(r.Context(), apiKey)
			if err == nil && user != nil {
				claims := &model.TokenClaims{
					UserID: user.ID,
					Email:  user.Email,
					Role:   user.Role,
				}
				ctx := context.WithValue(r.Context(), UserContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
	})
}

// RequireAdmin middleware checks for admin role
func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserFromContext(r.Context())
		if claims == nil || claims.Role != model.UserRoleAdmin {
			http.Error(w, `{"error": "forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GenerateToken creates a new JWT token
func (m *AuthMiddleware) GenerateToken(user *model.User) (string, int64, error) {
	expiresAt := time.Now().Add(time.Duration(m.expHours) * time.Hour)

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.jwtSecret)
	if err != nil {
		return "", 0, err
	}

	return tokenStr, expiresAt.Unix(), nil
}

// ValidateToken validates a JWT token and returns claims
func (m *AuthMiddleware) ValidateToken(tokenStr string) (*model.TokenClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &model.TokenClaims{
			UserID: claims["user_id"].(string),
			Email:  claims["email"].(string),
			Role:   model.UserRole(claims["role"].(string)),
		}, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// GetUserFromContext extracts user claims from context
func GetUserFromContext(ctx context.Context) *model.TokenClaims {
	claims, ok := ctx.Value(UserContextKey).(*model.TokenClaims)
	if !ok {
		return nil
	}
	return claims
}

// CORS middleware
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// JSON middleware sets JSON content type
func JSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Logger middleware logs requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		// Simple logging - could use structured logger
		println(r.Method, r.URL.Path, duration.String())
	})
}
