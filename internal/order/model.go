package order

import "time"

type OrderStatus string

const (
	StatusCreated   OrderStatus = "CREATED"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusCancelled OrderStatus = "CANCELLED"
)

type Order struct {
	ID         int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	CustomerID int64       `gorm:"column:customer_id;not null;index" json:"customerId" binding:"required,gt=0"`
	ProductID  int64       `gorm:"column:product_id;not null" json:"productId" binding:"required,gt=0"`
	Quantity   int         `gorm:"column:quantity;not null" json:"quantity" binding:"required,gt=0"`
	TotalPrice float64     `gorm:"column:total_price;type:numeric(12,2);not null" json:"totalPrice"`
	Status     OrderStatus `gorm:"column:status;type:varchar(20);not null;default:'CREATED';index" json:"status"`
	CreatedAt  time.Time   `gorm:"column:created_at;autoCreateTime;index" json:"createdAt"`
	UpdatedAt  time.Time   `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (Order) TableName() string {
	return "orders"
}

type CreateOrderRequest struct {
	CustomerID int64 `json:"customerId" binding:"required,gt=0"`
	ProductID  int64 `json:"productId" binding:"required,gt=0"`
	Quantity   int   `json:"quantity" binding:"required,gt=0"`
}

type UpdateStatusRequest struct {
	Status OrderStatus `json:"status" binding:"required"`
}
