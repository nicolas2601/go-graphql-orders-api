package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

func TestUserGet(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	_ = repo.Create(ctx, domain.User{ID: "u1", Email: "nico@example.com"})
	uc := usecase.NewUserUseCase(repo)

	t.Run("existing user is returned", func(t *testing.T) {
		u, err := uc.Get(ctx, "u1")
		if err != nil || u.Email != "nico@example.com" {
			t.Fatalf("got %+v, %v", u, err)
		}
	})

	t.Run("missing user returns ErrUserNotFound", func(t *testing.T) {
		if _, err := uc.Get(ctx, "ghost"); !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("got %v, want ErrUserNotFound", err)
		}
	})
}

func TestNewUserUseCasePanicsOnNilRepo(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on nil repository")
		}
	}()
	usecase.NewUserUseCase(nil)
}
