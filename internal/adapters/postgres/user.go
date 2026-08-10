package postgres

import (
	"errors"

	user "github.com/mthstanley/stockpot/internal/core"
)

type UserRepository struct{}

func (UserRepository) GetByID(id int64) (*user.User, error) {
	return nil, errors.New("method not implmented")
}
