package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/mthstanley/stockpot/db/sqlc"
	"github.com/mthstanley/stockpot/internal/core/auth"
)

func convertToAuthUserCredentialsDomainModel(authUser db.AuthUser) *auth.AuthUserCredentials {
	return &auth.AuthUserCredentials{
		ID:           &authUser.ID,
		Username:     authUser.Username,
		PasswordHash: authUser.PasswordHash,
		UserID:       authUser.AppUser.Int64,
	}
}

type AuthUserRepository struct {
	db *pgxpool.Pool
}

func NewAuthUserRepository(db *pgxpool.Pool) *AuthUserRepository {
	return &AuthUserRepository{db}
}

func (r AuthUserRepository) GetAuthUserCredentials(ctx context.Context, username string) (*auth.AuthUserCredentials, error) {
	queries := db.New(r.db)
	authUser, err := queries.GetAuthUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth user: %w", err)
	}
	return convertToAuthUserCredentialsDomainModel(authUser), nil
}

func (r AuthUserRepository) CreateAuthUserCredentials(ctx context.Context, authUser auth.AuthUserCredentials) (*auth.AuthUserCredentials, error) {
	queries := db.New(r.db)
	createdAuthUser, err := queries.CreateAuthUser(ctx, db.CreateAuthUserParams{
		Username:     authUser.Username,
		PasswordHash: authUser.PasswordHash,
		AppUser:      pgtype.Int8{Int64: authUser.UserID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create auth user: %w", err)
	}
	return convertToAuthUserCredentialsDomainModel(createdAuthUser), nil
}
