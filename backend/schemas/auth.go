package schemas

type ResponseIsInitialized struct {
	Initialized bool `json:"initialized"`
}

type RequestAuthRegister struct {
	Passphrase string `json:"passphrase"`
}

type ResponseAuthRegister struct {
	Recovery string `json:"recovery"`
}

type RequestAuthCompleteRegistration struct {
	Recovery string `json:"recovery"`
}

type RequestAuthLogin struct {
	Passphrase string `json:"passphrase"`
}

type ResponseAuthLogin struct {
	Token string `json:"token"`
}

type RequestAuthUpdatePassphrase struct {
	Passphrase string `json:"passphrase"`
}
