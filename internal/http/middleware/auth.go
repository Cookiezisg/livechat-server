package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"livechat-server/internal/auth"
	"livechat-server/internal/store"

	"github.com/google/uuid"
)

type contextKey string

const (
	userContextKey contextKey = "user"
)

func WithUser(ctx context.Context, user store.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (store.User, bool) {
	user, ok := ctx.Value(userContextKey).(store.User)
	return user, ok
}

type AuthMiddleware struct {
	Secret string
	Store  *store.Store
	TokenTTL time.Duration
}

func (a AuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		claims, err := auth.ParseToken(token, a.Secret)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		user, err := a.Store.GetUserByID(r.Context(), userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	return ""
}
