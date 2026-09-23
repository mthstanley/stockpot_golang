package recipe

import (
	"context"
	"time"

	"github.com/mthstanley/stockpot/internal/core/user"
)

const EntityType = "recipe"

type Unit struct {
	ID   *int64
	Name string
}

type Ingredient struct {
	ID   *int64
	Name string
}

type RecipeIngredient struct {
	ID          *int64
	RecipeID    *int64
	Ingredient  Ingredient
	Quantity    int
	Units       Unit
	Preparation string
}

type Step struct {
	ID          *int64
	RecipeID    *int64
	Ordinal     int
	Instruction string
}

type Recipe struct {
	ID            *int64
	Title         string
	Description   *string
	Author        user.User
	PrepTime      *time.Duration
	CookTime      *time.Duration
	InactiveTime  *time.Duration
	YieldQuantity int
	YieldUnits    Unit
	Ingredients   []RecipeIngredient
	Steps         []Step
}

type Repository interface {
	Get(ctx context.Context) ([]Recipe, error)
	GetByID(ctx context.Context, id int64) (*Recipe, error)
	Create(ctx context.Context, rec Recipe) (*Recipe, error)
	Update(ctx context.Context, rec Recipe) (*Recipe, error)
	DeleteByID(ctx context.Context, id int64) (*Recipe, error)
}

type Service interface {
	Get(ctx context.Context) ([]Recipe, error)
	GetByID(ctx context.Context, id int64) (*Recipe, error)
	Create(ctx context.Context, rec Recipe) (*Recipe, error)
	Update(ctx context.Context, rec Recipe) (*Recipe, error)
	DeleteByID(ctx context.Context, id int64) (*Recipe, error)
}

type DefaultService struct {
	repo Repository
}

func NewDefaultService(repo Repository) *DefaultService {
	return &DefaultService{repo}
}

func (s DefaultService) Get(ctx context.Context) ([]Recipe, error) {
	return s.repo.Get(ctx)
}

func (s DefaultService) GetByID(ctx context.Context, id int64) (*Recipe, error) {
	return s.repo.GetByID(ctx, id)
}

func (s DefaultService) Create(ctx context.Context, rec Recipe) (*Recipe, error) {
	return s.repo.Create(ctx, rec)
}

func (s DefaultService) Update(ctx context.Context, rec Recipe) (*Recipe, error) {
	return s.repo.Update(ctx, rec)
}

func (s DefaultService) DeleteByID(ctx context.Context, id int64) (*Recipe, error) {
	return s.repo.DeleteByID(ctx, id)
}
