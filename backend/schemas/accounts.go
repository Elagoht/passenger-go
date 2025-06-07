package schemas

import (
	"time"
)

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

func ToResponseAccountCard(account *ResponseAccountCard) ResponseAccountCard {
	return ResponseAccountCard{
		Id:         account.Id,
		Platform:   account.Platform,
		Identifier: account.Identifier,
		Favorite:   account.Favorite,
	}
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
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	AccessCount  int        `json:"accessCount"`
	Strength     int        `json:"strength"`
	LastAccessed *time.Time `json:"lastAccessed"`
}

type ResponseAccountPassphrase struct {
	Passphrase string `json:"passphrase"`
}
