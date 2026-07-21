package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// Tipos de token, guardados en el claim "typ" para que un access no sirva como refresh ni viceversa.
const (
	typeAccess  = "access"
	typeRefresh = "refresh"
)

var errWrongTokenType = errors.New("wrong token type")

// JWTService implementa domain.TokenService con JWT firmados con HMAC-SHA256.
type JWTService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// NewJWTService crea el servicio con el secreto, los TTL de access y refresh, y un reloj inyectable.
func NewJWTService(secret string, accessTTL, refreshTTL time.Duration, now func() time.Time) *JWTService {
	if now == nil {
		now = time.Now
	}
	return &JWTService{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL, now: now}
}

var _ domain.TokenService = (*JWTService)(nil)

// GenerateAccess emite un access token para el usuario.
func (s *JWTService) GenerateAccess(userID string) (string, error) {
	return s.generate(userID, typeAccess, s.accessTTL)
}

// GenerateRefresh emite un refresh token para el usuario.
func (s *JWTService) GenerateRefresh(userID string) (string, error) {
	return s.generate(userID, typeRefresh, s.refreshTTL)
}

func (s *JWTService) generate(userID, tokenType string, ttl time.Duration) (string, error) {
	now := s.now()
	claims := jwt.MapClaims{
		"sub": userID,
		"typ": tokenType,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// ParseAccess valida un access token y devuelve el userID.
func (s *JWTService) ParseAccess(token string) (string, error) {
	return s.parse(token, typeAccess)
}

// ParseRefresh valida un refresh token y devuelve el userID.
func (s *JWTService) ParseRefresh(token string) (string, error) {
	return s.parse(token, typeRefresh)
}

func (s *JWTService) parse(tokenStr, expectedType string) (string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims,
		func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil {
		return "", err
	}
	if typ, _ := claims["typ"].(string); typ != expectedType {
		return "", errWrongTokenType
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("token without subject")
	}
	return sub, nil
}
