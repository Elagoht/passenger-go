package schemas

import "time"

type RequestAccountsCreate struct {
	Platform   string `json:"platform" validate:"required"`
	Identifier string `json:"identifier" validate:"required"`
	Passphrase string `json:"passphrase" validate:"required"`
	Notes      string `json:"notes,omitempty"`
	Favorite   bool   `json:"favorite,omitempty"`
}

type ResponseAccountsCreate struct {
	Id string `json:"id"`
}

type RequestAccountsUpdate struct {
	Platform   string `json:"platform,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	Notes      string `json:"notes,omitempty"`
	Favorite   bool   `json:"favorite,omitempty"`
}

type ResponseAccountCard struct {
	Id         string `json:"id"`
	Platform   string `json:"platform"`
	Identifier string `json:"identifier"`
	Favorite   bool   `json:"favorite"`
}

type ResponseAccountCardList struct {
	Accounts []ResponseAccountCard `json:"accounts"`
}

type ResponseAccountDetails struct {
	Id           string     `json:"id"`
	Platform     string     `json:"platform"`
	Identifier   string     `json:"identifier"`
	Notes        string     `json:"notes,omitempty"`
	Favorite     bool       `json:"favorite"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	AccessCount  int        `json:"access_count"`
	Strength     int        `json:"strength"`
	LastAccessed *time.Time `json:"last_accessed"`
}

type ResponseAccountPassphrase struct {
	Passphrase string `json:"passphrase"`
}
