package middleware

import (
	"github.com/gorilla/mux"
)

const AdminIDKey contextKey = UserIDKey

func AdminAuthMiddleware() mux.MiddlewareFunc {
	return AuthMiddleware()
}
