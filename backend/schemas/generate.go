package schemas

type ResponseGenerate struct {
	Generated string `json:"generated"`
}

type RequestAlternate struct {
	Passphrase string `json:"passphrase" validate:"required"`
}

type ResponseAlternate struct {
	Alternative string `json:"alternative" `
}
