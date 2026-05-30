package auth

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Repo Repository
	JWT  JWTService
}

func NewService(repo Repository, jwtSvc JWTService) Service {
	return Service{Repo: repo, JWT: jwtSvc}
}

func (s Service) Register(ctx context.Context, req RegisterRequest) (TokenResponse, error) {
	email := normalizeEmail(req.Email)
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return TokenResponse{}, err
	}
	u, err := s.Repo.CreateUser(ctx, email, string(hash), req.Nickname)
	if err != nil {
		return TokenResponse{}, err
	}
	return s.tokenResponse(u)
}

func (s Service) Login(ctx context.Context, req LoginRequest) (TokenResponse, error) {
	email := normalizeEmail(req.Email)
	u, passwordHash, err := s.Repo.FindByEmail(ctx, email)
	if err != nil {
		return TokenResponse{}, err
	}
	if u.Status != "active" {
		return TokenResponse{}, errors.New("user is not active")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return TokenResponse{}, ErrInvalidCredentials
	}
	_ = s.Repo.TouchLastLogin(ctx, u.ID)
	return s.tokenResponse(u)
}

func (s Service) tokenResponse(u User) (TokenResponse, error) {
	token, expiresAt, err := s.JWT.Sign(u)
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: token, TokenType: "Bearer", ExpiresAt: expiresAt.Truncate(time.Second), User: u}, nil
}
