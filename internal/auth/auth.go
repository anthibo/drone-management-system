package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Authenticator struct {
	secret []byte
	ttl    time.Duration
}

func New(secret string, ttl time.Duration) *Authenticator {
	return &Authenticator{secret: []byte(secret), ttl: ttl}
}

func (a *Authenticator) IssueToken(name, role string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(a.ttl)
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   name,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := token.SignedString(a.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return str, exp, nil
}

func (a *Authenticator) ParseToken(tokenString string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return a.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func ExtractBearerToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

type ctxKey struct{}

func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	v := ctx.Value(ctxKey{})
	claims, ok := v.(*Claims)
	return claims, ok
}

