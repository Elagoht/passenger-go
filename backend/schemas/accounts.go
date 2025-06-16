package schemas

type RequestAccountsUpsert struct {
	Platform   string `json:"platform" validate:"required"`
	Identifier string `json:"identifier" validate:"required"`
	Passphrase string `json:"passphrase" validate:"required"`
	Url        string `json:"url" validate:"required"`
	Notes      string `json:"notes" validate:"omitempty"`
}

type ResponseAccount struct {
	Id         string `json:"id"`
	Platform   string `json:"platform"`
	Identifier string `json:"identifier"`
	Url        string `json:"url"`
	Notes      string `json:"notes"`
	Strength   int    `json:"strength"`
}

type ResponseAccountDetails struct {
	Id         string `json:"id"`
	Platform   string `json:"platform"`
	Identifier string `json:"identifier"`
	Url        string `json:"url"`
	Passphrase string `json:"passphrase"`
	Notes      string `json:"notes"`
	Strength   int    `json:"strength"`
}
