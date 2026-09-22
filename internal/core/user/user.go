package user

import (
	"context"
)

type User struct {
	ID   *int64
	Name string
}

func (u User) Equals(other User) bool {
	if u.ID == nil {
		return other.ID == nil
	}
	return *u.ID == *other.ID && u.Name == other.Name
}

const EntityType string = "user"

type Repository interface {
	GetByID(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, user User) (*User, error)
}

type Service interface {
	Get(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, user User) (*User, error)
}

type DefaultService struct {
	repo Repository
}

func NewDefaultService(repo Repository) *DefaultService {
	return &DefaultService{repo}
}

func (s DefaultService) Get(ctx context.Context, id int64) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s DefaultService) Create(ctx context.Context, user User) (*User, error) {
	return s.repo.Create(ctx, user)
}
