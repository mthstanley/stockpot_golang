package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mthstanley/stockpot/internal/core"
	"github.com/mthstanley/stockpot/internal/core/user"
	"golang.org/x/crypto/bcrypt"
)

type AuthUserCredentials struct {
	ID           *int64
	Username     string
	PasswordHash string
	UserID       int64
}

const EntityType string = "auth user"

var InvalidCredentialsError = errors.New("invalid credentials provided")
var CredentialValidationError = errors.New("credential validation failed")

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

type Claims struct {
	jwt.RegisteredClaims
}

type JWT struct {
	Token string
}

func (JWT) isCredential() {}

type Respository interface {
	GetAuthUserCredentials(ctx context.Context, username string) (*AuthUserCredentials, error)
	CreateAuthUserCredentials(ctx context.Context, authUser AuthUserCredentials) (*AuthUserCredentials, error)
}

type Service interface {
	Validate(ctx context.Context, credentials UserCredentials) (*AuthUser, error)
	CreateAuthUser(ctx context.Context, user user.User, credentials UsernameAndPassword) (*AuthUser, error)
	GenerateJWT(authUser AuthUser) (*JWT, error)
}

type DefaultService struct {
	repo             Respository
	userService      user.Service
	jwtSecret        string
	jwtAudience      string
	jwtExpiration    time.Duration
	jwtSigningMethod jwt.SigningMethod
}

func NewDefaultService(repo Respository, userService user.Service, jwtTokenSecret string) *DefaultService {
	return &DefaultService{
		repo:             repo,
		userService:      userService,
		jwtSecret:        jwtTokenSecret,
		jwtAudience:      "https://api.stockpot.com",
		jwtExpiration:    time.Duration(7 * 24 * time.Hour),
		jwtSigningMethod: jwt.SigningMethodHS256,
	}
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
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				return nil, fmt.Errorf("password is not valid: %w: %w", InvalidCredentialsError, err)
			}
			return nil, fmt.Errorf("unable to validate password: %w: %w", CredentialValidationError, err)
		}

		if errResult != nil {
			aerr, ok := errors.AsType[*core.EntityNotFound](errResult)
			authUserNotFound := ok && aerr.Type == EntityType
			if authUserNotFound {
				return nil, fmt.Errorf("no auth user with given username: %w: %w", InvalidCredentialsError, errResult)
			}
			return nil, fmt.Errorf("unable to fetch auth user: %w: %w", CredentialValidationError, errResult)
		}

		return &AuthUser{Username: authUser.Username, User: *userResult}, nil
	case JWT:
		token, err := jwt.ParseWithClaims(c.Token, &Claims{}, func(t *jwt.Token) (any, error) {
			return []byte(s.jwtSecret), nil
		}, jwt.WithValidMethods([]string{s.jwtSigningMethod.Alg()}))

		if err != nil {
			return nil, fmt.Errorf("unable to parse and validate jwt: %w: %w", InvalidCredentialsError, err)
		} else if claims, ok := token.Claims.(*Claims); ok {
			authUser, err := s.repo.GetAuthUserCredentials(ctx, claims.Subject)
			if err != nil {
				if _, ok := errors.AsType[*core.EntityNotFound](err); ok {
					return nil, fmt.Errorf("no matching auth user for jwt subject: %w: %w", InvalidCredentialsError, err)
				}
				return nil, fmt.Errorf("unable to fetch auth user by jwt subject: %w: %w", CredentialValidationError, err)
			}

			userResult, err := s.userService.Get(ctx, authUser.UserID)
			if err != nil {
				return nil, fmt.Errorf("unable to fetch user for auth user: %w: %w", CredentialValidationError, err)
			}

			return &AuthUser{Username: authUser.Username, User: *userResult}, nil
		} else {
			return nil, fmt.Errorf("unknown claims type, cannot proceed: %w", InvalidCredentialsError)
		}
	default:
		return nil, fmt.Errorf("could not validate unsupported credential type %T: %w", credentials, InvalidCredentialsError)
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

func (s DefaultService) GenerateJWT(authUser AuthUser) (*JWT, error) {
	claims := Claims{
		jwt.RegisteredClaims{
			Audience:  []string{s.jwtAudience},
			Subject:   authUser.Username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(s.jwtSigningMethod, claims)

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate jwt: %w", err)
	}

	return &JWT{Token: tokenString}, nil
}
