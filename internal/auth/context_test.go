package auth

import (
	"context"
	"testing"

	"github.com/safari3308/adguard-portal/internal/domain"
)

func TestContextUser(t *testing.T) {
	ctx := context.Background()

	// Empty context
	if u := UserFromContext(ctx); u != nil {
		t.Errorf("expected nil user from empty context, got %+v", u)
	}

	// Context with wrong type
	wrongCtx := context.WithValue(ctx, userContextKey, "not-a-user")
	if u := UserFromContext(wrongCtx); u != nil {
		t.Errorf("expected nil user when context has wrong type, got %+v", u)
	}

	// Context with valid domain.User
	expectedUser := &domain.User{
		ID:    "user-123",
		Email: "test@example.com",
		Name:  "Test User",
	}
	userCtx := WithUser(ctx, expectedUser)
	gotUser := UserFromContext(userCtx)
	if gotUser == nil || gotUser.ID != expectedUser.ID {
		t.Errorf("expected user %+v, got %+v", expectedUser, gotUser)
	}
}
