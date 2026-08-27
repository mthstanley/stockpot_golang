package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/mthstanley/stockpot/internal/core/user"
	"golang.org/x/crypto/bcrypt"
)

type AuthUserCredentials struct {
	ID           *int64
	Username     string
	PasswordHash string
	UserID       int64
}

type AuthUser struct {
	Username string
	user.User
}

type UserCredentials interface {
	isCredential()
}

type UsernameAndPassword struct {
	Username string
	Password string
}

func (UsernameAndPassword) isCredential() {}

type Respository interface {
	GetAuthUserCredentials(ctx context.Context, username string) (*AuthUserCredentials, error)
	CreateAuthUserCredentials(ctx context.Context, authUser AuthUserCredentials) (*AuthUserCredentials, error)
}

type Service interface {
	Validate(ctx context.Context, credentials UserCredentials) (*AuthUser, error)
	CreateAuthUser(ctx context.Context, user user.User, credentials UsernameAndPassword) (*AuthUser, error)
}

type DefaultService struct {
	repo        Respository
	userService user.Service
}

func NewDefaultService(repo Respository, userService user.Service) *DefaultService {
	return &DefaultService{repo, userService}
}

func (s DefaultService) Validate(ctx context.Context, credentials UserCredentials) (*AuthUser, error) {
	switch c := credentials.(type) {
	case UsernameAndPassword:
		// we should always do some hash comparison to avoid timing attacks
		expectedPasswordHash := "$2a$10$hEKXCteotpUJy.WG3aid9ugM1vDkiB.e.P3R2Tc/GyY6C2k4yVCHG"
		authUser, errResult := s.repo.GetAuthUserCredentials(ctx, c.Username)
		var userResult *user.User
		if errResult == nil {
			expectedPasswordHash = authUser.PasswordHash
			userResult, errResult = s.userService.Get(ctx, authUser.UserID)
		}
		err := bcrypt.CompareHashAndPassword([]byte(expectedPasswordHash), []byte(c.Password))
		if err != nil {
			return nil, fmt.Errorf("password is not valid: %w", err)
		}

		if errResult != nil {
			return nil, fmt.Errorf("unable to fetch auth user: %w", errResult)
		}

		return &AuthUser{Username: authUser.Username, User: *userResult}, nil
	default:
		return nil, fmt.Errorf("could not validate unsupported credential type %T", credentials)
	}
}

func (s DefaultService) CreateAuthUser(ctx context.Context, user user.User, credentials UsernameAndPassword) (*AuthUser, error) {
	if user.ID == nil {
		return nil, errors.New("provided user is missing ID")
	}

	if _, err := s.userService.Get(ctx, *user.ID); err != nil {
		return nil, fmt.Errorf("unable to validate provided user: %w", err)
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(credentials.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("unable to hash given password %w", err)
	}

	authUserCredentials, err := s.repo.CreateAuthUserCredentials(ctx, AuthUserCredentials{
		ID:           nil,
		Username:     credentials.Username,
		PasswordHash: string(bytes),
		UserID:       *user.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to perist auth user: %w", err)
	}

	return &AuthUser{
		Username: authUserCredentials.Username,
		User:     user,
	}, nil
}
