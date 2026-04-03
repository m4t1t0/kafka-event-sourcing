package order

import "time"

// Order is the GORM read model — the projection of order events.
type Order struct {
	ID          string  `gorm:"primaryKey;type:varchar(36)" json:"id"`
	ClientID    string  `gorm:"type:varchar(36);index;not null" json:"client_id"`
	ClientName  string  `gorm:"type:varchar(255)" json:"client_name"`
	Status      string  `gorm:"type:varchar(20);not null;default:'placed'" json:"status"`
	Total       float64 `gorm:"type:decimal(12,2);not null" json:"total"`
	ConfirmedBy string  `gorm:"type:varchar(255)" json:"confirmed_by,omitempty"`
	CancelReason string `gorm:"type:varchar(500)" json:"cancel_reason,omitempty"`
	Version     int       `gorm:"default:0" json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
}

// OrderItem is a line item within an order projection.
type OrderItem struct {
	ID        uint    `gorm:"primaryKey" json:"-"`
	OrderID   string  `gorm:"type:varchar(36);index;not null" json:"order_id"`
	ProductID string  `gorm:"type:varchar(255);not null" json:"product_id"`
	Name      string  `gorm:"type:varchar(255);not null" json:"name"`
	Quantity  int     `gorm:"not null" json:"quantity"`
	Price     float64 `gorm:"type:decimal(12,2);not null" json:"price"`
}
