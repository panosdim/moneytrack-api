package models

type Expense struct {
	ID       uint    `gorm:"primary_key;autoIncrement" json:"id"`
	UserID   uint    `gorm:"not null" json:"-"`
	Amount   float64 `gorm:"not null" json:"amount"`
	Category uint    `gorm:"not null" json:"count"`
	Comment  string  `gorm:"not null" json:"comment"`
	Date     string  `gorm:"not null" json:"date"`
}
