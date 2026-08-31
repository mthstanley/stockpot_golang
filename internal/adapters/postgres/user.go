package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/mthstanley/stockpot/db/sqlc"
	"github.com/mthstanley/stockpot/internal/core"
	user "github.com/mthstanley/stockpot/internal/core/user"
)

func convertToUserDomainModel(appUser db.AppUser) *user.User {
	return &user.User{
		ID:   &appUser.ID,
		Name: appUser.Name,
	}
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db}
}

func (r UserRepository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	queries := db.New(r.db)
	appUser, err := queries.GetAppUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &core.EntityNotFound{Type: user.EntityType, Ident: strconv.FormatInt(id, 10), Err: err}
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}
	return convertToUserDomainModel(appUser), nil
}

func (r UserRepository) Create(ctx context.Context, user user.User) (*user.User, error) {
	queries := db.New(r.db)
	appUser, err := queries.CreateAppUser(ctx, user.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return convertToUserDomainModel(appUser), nil
}
