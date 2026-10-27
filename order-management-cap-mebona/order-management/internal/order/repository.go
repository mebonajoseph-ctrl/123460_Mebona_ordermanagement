package order

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

var ErrOrderNotFound = errors.New("order not found")

type Repository interface {
	Create(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id int64) (*Order, error)
	List(ctx context.Context) ([]*Order, error)
	UpdateStatus(ctx context.Context, id int64, status OrderStatus) error
	Ping(ctx context.Context) error
}

type gormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, o *Order) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *gormRepository) GetByID(ctx context.Context, id int64) (*Order, error) {
	var o Order
	err := r.db.WithContext(ctx).First(&o, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (r *gormRepository) List(ctx context.Context) ([]*Order, error) {
	var orders []*Order
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&orders).Error
	return orders, err
}

func (r *gormRepository) UpdateStatus(ctx context.Context, id int64, status OrderStatus) error {
	res := r.db.WithContext(ctx).Model(&Order{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (r *gormRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// MemoryRepository for testing and concurrency tests without external DB requirement
type MemoryRepository struct {
	mu     sync.RWMutex
	orders map[int64]*Order
	nextID int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		orders: make(map[int64]*Order),
		nextID: 1,
	}
}

func (m *MemoryRepository) Create(ctx context.Context, o *Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	o.ID = m.nextID
	m.nextID++
	o.CreatedAt = now
	o.UpdatedAt = now

	cloned := *o
	m.orders[o.ID] = &cloned
	return nil
}

func (m *MemoryRepository) GetByID(ctx context.Context, id int64) (*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	o, ok := m.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	cloned := *o
	return &cloned, nil
}

func (m *MemoryRepository) List(ctx context.Context) ([]*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Order, 0, len(m.orders))
	for _, o := range m.orders {
		cloned := *o
		list = append(list, &cloned)
	}
	return list, nil
}

func (m *MemoryRepository) UpdateStatus(ctx context.Context, id int64, status OrderStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	o, ok := m.orders[id]
	if !ok {
		return ErrOrderNotFound
	}
	o.Status = status
	o.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemoryRepository) Ping(ctx context.Context) error {
	return nil
}
