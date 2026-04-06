package models

type Income struct {
	ID      uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID  uint    `gorm:"not null" json:"-"`
	Amount  float64 `gorm:"not null" json:"amount"`
	Comment string  `gorm:"not null" json:"comment"`
	Date    string  `gorm:"not null" json:"date"`
}
