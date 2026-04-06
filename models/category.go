package models

type Category struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID   uint   `gorm:"not null" json:"-"`
	Category string `gorm:"not null" json:"category"`
	Count    uint   `gorm:"not null" json:"count"`
}
