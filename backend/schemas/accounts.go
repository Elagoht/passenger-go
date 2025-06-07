package schemas

import (
	"time"
)

type RequestAccountsCreate struct {
	Platform   string `json:"platform" validate:"required"`
	Identifier string `json:"identifier" validate:"required"`
	Passphrase string `json:"passphrase" validate:"required"`
	Url        string `json:"url" validate:"required"`
	Notes      string `json:"notes" validate:"omitempty"`
	Favorite   bool   `json:"favorite" validate:"omitempty"`
}

type ResponseAccountsCreate struct {
	Id string `json:"id"`
}

type RequestAccountsUpdate struct {
	Platform   string `json:"platform" validate:"required"`
	Identifier string `json:"identifier" validate:"required"`
	Passphrase string `json:"passphrase" validate:"required"`
	Url        string `json:"url" validate:"required"`
	Notes      string `json:"notes" validate:"omitempty"`
	Favorite   bool   `json:"favorite" validate:"omitempty"`
}

type ResponseAccountCard struct {
	Id         string `json:"id"`
	Platform   string `json:"platform"`
	Identifier string `json:"identifier"`
	Url        string `json:"url"`
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
	Url          string     `json:"url"`
	Notes        string     `json:"notes"`
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
