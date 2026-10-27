package product

import "time"

type Product struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	Price     float64   `gorm:"column:price;type:numeric(12,2);not null" json:"price"`
	Stock     int       `gorm:"column:stock;not null" json:"stock"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (Product) TableName() string {
	return "products"
}
