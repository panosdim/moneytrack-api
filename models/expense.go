package models

import (
	"database/sql/driver"
	"time"
)

type Date time.Time

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02") + `"`), nil
}

func (d *Date) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	t, err := time.Parse(`"2006-01-02"`, string(data))
	if err != nil {
		return err
	}
	*d = Date(t)
	return nil
}

// Value implements the driver.Valuer interface for database/sql
func (d Date) Value() (driver.Value, error) {
	return time.Time(d), nil
}

// Scan implements the sql.Scanner interface for database/sql
func (d *Date) Scan(value any) error {
	if value == nil {
		*d = Date(time.Time{})
		return nil
	}

	if t, ok := value.(time.Time); ok {
		*d = Date(t)
		return nil
	}

	// If it's a string, parse it
	if s, ok := value.(string); ok {
		t, err := time.Parse("2006-01-02 15:04:05", s)
		if err != nil {
			t, err = time.Parse("2006-01-02", s)
			if err != nil {
				return err
			}
		}
		*d = Date(t)
		return nil
	}

	return nil
}

type Expense struct {
	ID       uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID   uint    `gorm:"not null" json:"-"`
	Amount   float64 `gorm:"not null" json:"amount"`
	Category uint    `gorm:"not null" json:"category"`
	Comment  string  `gorm:"not null" json:"comment"`
	Date     Date    `gorm:"not null" json:"date"`
}
