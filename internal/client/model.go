package client

import "time"

// Client is the GORM read model — the projection of client events.
type Client struct {
	ID        string `gorm:"primaryKey;type:varchar(36)"`
	Name      string `gorm:"type:varchar(255);not null"`
	Email     string `gorm:"type:varchar(255);uniqueIndex"`
	Phone     string `gorm:"type:varchar(50)"`
	Version   int    `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"` // Soft delete
}
