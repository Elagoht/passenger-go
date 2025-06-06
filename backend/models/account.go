package models

import "time"

type Account struct {
	Id           string
	Platform     string
	Identifier   string
	Passphrase   string
	Notes        string
	Favorite     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	AccessCount  int
	Strength     int
	LastAccessed *time.Time
}
