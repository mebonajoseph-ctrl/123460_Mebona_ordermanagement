package product

import (
	"context"
)

type Service interface {
	GetProduct(ctx context.Context, id int64) (*Product, error)
	GetPrice(ctx context.Context, id int64) (float64, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetProduct(ctx context.Context, id int64) (*Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetPrice(ctx context.Context, id int64) (float64, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return 0, err
	}
	return p.Price, nil
}
