package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	Secret []byte
	Issuer string
	TTL    time.Duration
}

func NewJWTService(secret, issuer string, ttlHours int) (JWTService, error) {
	if err := validateSecret(secret); err != nil {
		return JWTService{}, err
	}
	if ttlHours <= 0 {
		ttlHours = 24
	}
	return JWTService{Secret: []byte(secret), Issuer: issuer, TTL: time.Duration(ttlHours) * time.Hour}, nil
}

func (s JWTService) Sign(user User) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.TTL)
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.Issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.Secret)
	return signed, expiresAt, err
}

func (s JWTService) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected jwt signing method")
		}
		return s.Secret, nil
	}, jwt.WithIssuer(s.Issuer))
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.UserID == "" {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
