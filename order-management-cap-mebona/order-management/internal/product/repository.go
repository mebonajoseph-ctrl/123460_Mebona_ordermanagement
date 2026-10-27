package product

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

var ErrProductNotFound = errors.New("product not found")

type Repository interface {
	GetByID(ctx context.Context, id int64) (*Product, error)
	Create(ctx context.Context, p *Product) error
	List(ctx context.Context) ([]*Product, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) GetByID(ctx context.Context, id int64) (*Product, error) {
	var p Product
	err := r.db.WithContext(ctx).First(&p, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) Create(ctx context.Context, p *Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *gormRepository) List(ctx context.Context) ([]*Product, error) {
	var list []*Product
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

// MemoryRepository for testing/standalone execution
type MemoryRepository struct {
	mu       sync.RWMutex
	products map[int64]*Product
}

func NewMemoryRepository() *MemoryRepository {
	m := &MemoryRepository{
		products: make(map[int64]*Product),
	}
	// Seed demo catalog products
	m.products[501] = &Product{ID: 501, Name: "Ergonomic Mechanical Keyboard", Price: 150.0, Stock: 50, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.products[502] = &Product{ID: 502, Name: "4K IPS Monitor", Price: 400.0, Stock: 20, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	return m
}

func (m *MemoryRepository) GetByID(ctx context.Context, id int64) (*Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.products[id]
	if !ok {
		return nil, ErrProductNotFound
	}
	cloned := *p
	return &cloned, nil
}

func (m *MemoryRepository) Create(ctx context.Context, p *Product) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	cloned := *p
	m.products[p.ID] = &cloned
	return nil
}

func (m *MemoryRepository) List(ctx context.Context) ([]*Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*Product
	for _, p := range m.products {
		cloned := *p
		list = append(list, &cloned)
	}
	return list, nil
}
