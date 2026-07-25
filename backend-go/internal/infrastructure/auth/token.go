package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrMissingSecret = errors.New("jwt secret is required")
	ErrInvalidToken  = errors.New("invalid token")
)

type TokenManager struct {
	secret   []byte
	now      func() time.Time
	tokenTTL time.Duration
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrMissingSecret
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("token duration must be positive")
	}

	return &TokenManager{
		secret:   []byte(secret),
		now:      time.Now,
		tokenTTL: ttl,
	}, nil
}

func (tm *TokenManager) GenerateToken(userID uuid.UUID) (string, error) {
	if len(tm.secret) == 0 {
		return "", ErrMissingSecret
	}

	now := tm.now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(tm.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

func (tm *TokenManager) VerifyToken(tokenString string) (uuid.UUID, error) {
	if len(tm.secret) == 0 {
		return uuid.Nil, ErrMissingSecret
	}

	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return uuid.Nil, ErrInvalidToken
	}

	claims := new(Claims)

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			return tm.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(tm.now),
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	if claims.UserID == uuid.Nil {
		subjectID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return uuid.Nil, fmt.Errorf("%w: invalid subject", ErrInvalidToken)
		}

		return subjectID, nil
	}

	return claims.UserID, nil
}
