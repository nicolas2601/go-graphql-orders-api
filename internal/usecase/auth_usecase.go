// Package usecase contiene la logica de aplicacion. Depende de las interfaces del dominio,
// nunca de implementaciones concretas de infraestructura ni de la capa de delivery.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// TokenPair es el par de tokens que emite la autenticacion.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// dummyPassword se hashea una vez al construir el caso de uso para tener un hash valido contra el
// cual comparar en el login cuando el email no existe, e igualar asi el tiempo de respuesta.
const dummyPassword = "timing-attack-mitigation-dummy"

// AuthUseCase orquesta el registro, login y refresh de usuarios.
type AuthUseCase struct {
	users     domain.UserRepository
	hasher    domain.PasswordHasher
	tokens    domain.TokenService
	newID     func() string
	now       func() time.Time
	dummyHash string
}

// NewAuthUseCase construye el caso de uso con sus dependencias inyectadas.
func NewAuthUseCase(
	users domain.UserRepository,
	hasher domain.PasswordHasher,
	tokens domain.TokenService,
	newID func() string,
	now func() time.Time,
) *AuthUseCase {
	if users == nil || hasher == nil || tokens == nil || newID == nil || now == nil {
		panic("usecase: auth dependencies must not be nil")
	}
	uc := &AuthUseCase{users: users, hasher: hasher, tokens: tokens, newID: newID, now: now}
	uc.dummyHash, _ = hasher.Hash(dummyPassword)
	return uc
}

// Register valida email y contrasena, crea el usuario con la contrasena hasheada y emite tokens.
func (uc *AuthUseCase) Register(ctx context.Context, email, password string) (TokenPair, domain.User, error) {
	email = domain.NormalizeEmail(email)
	if err := domain.ValidateEmail(email); err != nil {
		return TokenPair{}, domain.User{}, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return TokenPair{}, domain.User{}, err
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	user, err := domain.NewUser(uc.newID(), email, hash, uc.now())
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	// Se emiten los tokens antes de persistir: si la generacion falla, no queda un usuario
	// creado sin poder devolverle sus tokens. Si Create falla (email duplicado), los tokens
	// simplemente se descartan.
	pair, err := uc.issueTokens(user.ID)
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return TokenPair{}, domain.User{}, err
	}
	return pair, user, nil
}

// Login autentica por email y contrasena y emite tokens. No distingue entre email inexistente y
// contrasena incorrecta: ambos devuelven ErrInvalidCredentials.
func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (TokenPair, domain.User, error) {
	email = domain.NormalizeEmail(email)
	user, err := uc.users.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrUserNotFound) {
		// Comparacion dummy para igualar el tiempo de respuesta: sin esto, un email inexistente
		// responderia mas rapido (no llega a bcrypt) y permitiria enumerar emails registrados.
		_ = uc.hasher.Compare(uc.dummyHash, password)
		return TokenPair{}, domain.User{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	if err := uc.hasher.Compare(user.PasswordHash, password); err != nil {
		return TokenPair{}, domain.User{}, domain.ErrInvalidCredentials
	}

	pair, err := uc.issueTokens(user.ID)
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	return pair, user, nil
}

// Refresh emite un par de tokens nuevo a partir de un refresh token valido de un usuario existente.
func (uc *AuthUseCase) Refresh(ctx context.Context, refreshToken string) (TokenPair, domain.User, error) {
	userID, err := uc.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return TokenPair{}, domain.User{}, domain.ErrUnauthenticated
	}
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return TokenPair{}, domain.User{}, domain.ErrUnauthenticated
	}

	pair, err := uc.issueTokens(user.ID)
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	return pair, user, nil
}

func (uc *AuthUseCase) issueTokens(userID string) (TokenPair, error) {
	access, err := uc.tokens.GenerateAccess(userID)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := uc.tokens.GenerateRefresh(userID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
