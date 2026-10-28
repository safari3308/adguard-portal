package auth

import (
	"context"

	"github.com/safari3308/adguard-portal/internal/domain"
)

type contextKey string

const userContextKey = contextKey("auth_user")

func WithUser(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) *domain.User {
	if val := ctx.Value(userContextKey); val != nil {
		if u, ok := val.(*domain.User); ok {
			return u
		}
	}
	return nil
}
