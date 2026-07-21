package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

// --- Dobles de prueba ---

type fakeUserRepo struct {
	byID    map[string]domain.User
	byEmail map[string]domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[string]domain.User{}, byEmail: map[string]domain.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, u domain.User) error {
	if _, ok := f.byEmail[u.Email]; ok {
		return domain.ErrEmailAlreadyRegistered
	}
	f.byID[u.ID] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) FindByIDs(_ context.Context, ids []string) (map[string]domain.User, error) {
	out := map[string]domain.User{}
	for _, id := range ids {
		if u, ok := f.byID[id]; ok {
			out[id] = u
		}
	}
	return out, nil
}

// fakeHasher hashea de forma reversible para los tests.
type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) { return "hashed:" + password, nil }
func (fakeHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return errors.New("mismatch")
	}
	return nil
}

// fakeTokens codifica el userID y el tipo en el token para poder verificarlos en los tests.
type fakeTokens struct{}

func (fakeTokens) GenerateAccess(userID string) (string, error)  { return "access:" + userID, nil }
func (fakeTokens) GenerateRefresh(userID string) (string, error) { return "refresh:" + userID, nil }
func (fakeTokens) ParseAccess(token string) (string, error) {
	return parseTyped(token, "access:")
}
func (fakeTokens) ParseRefresh(token string) (string, error) {
	return parseTyped(token, "refresh:")
}
func parseTyped(token, prefix string) (string, error) {
	if !strings.HasPrefix(token, prefix) {
		return "", errors.New("wrong token type")
	}
	return strings.TrimPrefix(token, prefix), nil
}

func newAuthUseCase(repo domain.UserRepository) *usecase.AuthUseCase {
	return usecase.NewAuthUseCase(repo, fakeHasher{}, fakeTokens{},
		func() string { return "user-1" },
		func() time.Time { return time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC) },
	)
}

// --- Tests ---

func TestRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("registers a user with hashed password and issues tokens", func(t *testing.T) {
		repo := newFakeUserRepo()
		uc := newAuthUseCase(repo)

		pair, user, err := uc.Register(ctx, "Nico@Example.com", "secret123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.Email != "nico@example.com" {
			t.Fatalf("email not normalized: %q", user.Email)
		}
		stored := repo.byID["user-1"]
		if stored.PasswordHash != "hashed:secret123" {
			t.Fatalf("password not hashed: %q", stored.PasswordHash)
		}
		if pair.AccessToken != "access:user-1" || pair.RefreshToken != "refresh:user-1" {
			t.Fatalf("unexpected tokens: %+v", pair)
		}
	})

	t.Run("duplicate email returns ErrEmailAlreadyRegistered and issues no token", func(t *testing.T) {
		repo := newFakeUserRepo()
		uc := newAuthUseCase(repo)
		_, _, _ = uc.Register(ctx, "nico@example.com", "secret123")

		pair, _, err := uc.Register(ctx, "nico@example.com", "secret123")
		if !errors.Is(err, domain.ErrEmailAlreadyRegistered) {
			t.Fatalf("got %v, want ErrEmailAlreadyRegistered", err)
		}
		if pair.AccessToken != "" {
			t.Fatalf("no token should be issued on failure")
		}
	})

	t.Run("invalid password returns ErrInvalidPassword and persists nothing", func(t *testing.T) {
		repo := newFakeUserRepo()
		uc := newAuthUseCase(repo)

		if _, _, err := uc.Register(ctx, "nico@example.com", "short"); !errors.Is(err, domain.ErrInvalidPassword) {
			t.Fatalf("got %v, want ErrInvalidPassword", err)
		}
		if len(repo.byID) != 0 {
			t.Fatalf("nothing should have been persisted")
		}
	})

	t.Run("invalid email returns ErrInvalidEmail", func(t *testing.T) {
		repo := newFakeUserRepo()
		uc := newAuthUseCase(repo)
		if _, _, err := uc.Register(ctx, "not-an-email", "secret123"); !errors.Is(err, domain.ErrInvalidEmail) {
			t.Fatalf("got %v, want ErrInvalidEmail", err)
		}
	})
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	seed := func() *fakeUserRepo {
		repo := newFakeUserRepo()
		_, _, _ = newAuthUseCase(repo).Register(ctx, "nico@example.com", "secret123")
		return repo
	}

	t.Run("valid credentials issue tokens", func(t *testing.T) {
		uc := newAuthUseCase(seed())
		pair, _, err := uc.Login(ctx, "Nico@Example.com", "secret123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pair.AccessToken != "access:user-1" {
			t.Fatalf("unexpected access token: %q", pair.AccessToken)
		}
	})

	t.Run("wrong password returns ErrInvalidCredentials", func(t *testing.T) {
		uc := newAuthUseCase(seed())
		if _, _, err := uc.Login(ctx, "nico@example.com", "wrongpass1"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("got %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("unknown email returns ErrInvalidCredentials", func(t *testing.T) {
		uc := newAuthUseCase(newFakeUserRepo())
		if _, _, err := uc.Login(ctx, "ghost@example.com", "secret123"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("got %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestRefresh(t *testing.T) {
	ctx := context.Background()

	t.Run("valid refresh token issues a new pair", func(t *testing.T) {
		repo := newFakeUserRepo()
		uc := newAuthUseCase(repo)
		pair, _, _ := uc.Register(ctx, "nico@example.com", "secret123")

		newPair, _, err := uc.Refresh(ctx, pair.RefreshToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if newPair.AccessToken != "access:user-1" {
			t.Fatalf("unexpected access token: %q", newPair.AccessToken)
		}
	})

	t.Run("access token used as refresh returns ErrUnauthenticated", func(t *testing.T) {
		repo := newFakeUserRepo()
		uc := newAuthUseCase(repo)
		pair, _, _ := uc.Register(ctx, "nico@example.com", "secret123")

		if _, _, err := uc.Refresh(ctx, pair.AccessToken); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("got %v, want ErrUnauthenticated", err)
		}
	})

	t.Run("refresh for a deleted user returns ErrUnauthenticated", func(t *testing.T) {
		uc := newAuthUseCase(newFakeUserRepo()) // repo vacio: el usuario del token no existe
		if _, _, err := uc.Refresh(ctx, "refresh:user-1"); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("got %v, want ErrUnauthenticated", err)
		}
	})
}

// errHasher y errTokens fuerzan fallos de infraestructura para cubrir esos caminos de error.
type errHasher struct{}

func (errHasher) Hash(string) (string, error) { return "", errors.New("hash failed") }
func (errHasher) Compare(_, _ string) error   { return nil }

type errTokens struct{}

func (errTokens) GenerateAccess(string) (string, error)  { return "", errors.New("token failed") }
func (errTokens) GenerateRefresh(string) (string, error) { return "", errors.New("token failed") }
func (errTokens) ParseAccess(string) (string, error)     { return "", errors.New("token failed") }
func (errTokens) ParseRefresh(string) (string, error)    { return "", errors.New("token failed") }

func TestRegisterPropagatesInfraErrors(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC) }
	id := func() string { return "user-1" }

	t.Run("hasher failure propagates", func(t *testing.T) {
		uc := usecase.NewAuthUseCase(newFakeUserRepo(), errHasher{}, fakeTokens{}, id, now)
		if _, _, err := uc.Register(ctx, "nico@example.com", "secret123"); err == nil {
			t.Fatal("expected hasher error to propagate")
		}
	})

	t.Run("token generation failure propagates", func(t *testing.T) {
		uc := usecase.NewAuthUseCase(newFakeUserRepo(), fakeHasher{}, errTokens{}, id, now)
		if _, _, err := uc.Register(ctx, "nico@example.com", "secret123"); err == nil {
			t.Fatal("expected token error to propagate")
		}
	})
}

func TestNewAuthUseCasePanicsOnNilDependency(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when a dependency is nil")
		}
	}()
	usecase.NewAuthUseCase(nil, nil, nil, nil, nil)
}
