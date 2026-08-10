package user

type User struct {
	ID   *int64
	Name string
}

type Repository interface {
	GetByID(id int64) (*User, error)
}

type Service interface {
	Get(id int64) (*User, error)
}

type DefaultService struct {
	Repo Repository
}

func (s DefaultService) Get(id int64) (*User, error) {
	return s.Repo.GetByID(id)
}
