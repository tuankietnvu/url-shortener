package model

import "time"

type URL struct {
	ID        uint       `gorm:"primaryKey"`
	ShortID   string     `gorm:"uniqueIndex;size:32;not null"`
	LongURL   string     `gorm:"type:text;not null"`
	Clicks    int        `gorm:"not null;default:0"`
	CreatedAt time.Time  `gorm:"not null;autoCreateTime"`
	ExpiredAt *time.Time `gorm:"index"`
}

