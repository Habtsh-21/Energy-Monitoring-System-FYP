package middleware

import (
	"context"
	"energy-monitoring-system/internal/auth"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const RoleKey contextKey = "role"

func AuthMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			token, err := auth.ValidateJWT(parts[1])
			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			userIDStr, ok := claims["user_id"].(string)
			if !ok {
				http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
				return
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				http.Error(w, "Invalid user ID format", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			if role, ok := claims["role"].(string); ok {
				ctx = context.WithValue(ctx, RoleKey, role)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
