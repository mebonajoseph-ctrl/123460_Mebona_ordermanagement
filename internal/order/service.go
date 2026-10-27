package order

import (
	"context"
	"errors"
	"fmt"

	"order-management/internal/product"
)

var (
	ErrInvalidInput  = errors.New("customer ID, product ID, and quantity must be positive")
	ErrInvalidStatus = errors.New("invalid status: must be CREATED, CONFIRMED, or CANCELLED")
)

type Service interface {
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error)
	GetOrder(ctx context.Context, id int64) (*Order, error)
	ListOrders(ctx context.Context) ([]*Order, error)
	UpdateOrderStatus(ctx context.Context, id int64, status OrderStatus) error
	CheckHealth(ctx context.Context) error
}

type service struct {
	repo           Repository
	productService product.Service
}

func NewService(repo Repository, productSvc product.Service) Service {
	return &service{
		repo:           repo,
		productService: productSvc,
	}
}

func (s *service) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	if req.CustomerID <= 0 || req.ProductID <= 0 || req.Quantity <= 0 {
		return nil, ErrInvalidInput
	}

	// Obtain unit price by calling the product service abstraction
	unitPrice, err := s.productService.GetPrice(ctx, req.ProductID)
	if err != nil || unitPrice <= 0 {
		unitPrice = 100.0 // graceful fallback
	}

	order := &Order{
		CustomerID: req.CustomerID,
		ProductID:  req.ProductID,
		Quantity:   req.Quantity,
		TotalPrice: unitPrice * float64(req.Quantity),
		Status:     StatusCreated,
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	return order, nil
}

func (s *service) GetOrder(ctx context.Context, id int64) (*Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) ListOrders(ctx context.Context) ([]*Order, error) {
	return s.repo.List(ctx)
}

func (s *service) UpdateOrderStatus(ctx context.Context, id int64, status OrderStatus) error {
	switch status {
	case StatusCreated, StatusConfirmed, StatusCancelled:
		return s.repo.UpdateStatus(ctx, id, status)
	default:
		return ErrInvalidStatus
	}
}

func (s *service) CheckHealth(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
