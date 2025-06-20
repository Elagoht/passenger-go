package schemas

type ResponseIsInitialized struct {
	Initialized bool `json:"initialized"`
}

type RequestAuthRegister struct {
	Passphrase string `json:"passphrase" validate:"required,min=12,max=128"`
}

type ResponseAuthRegister struct {
	Recovery string `json:"recovery"`
}

type RequestAuthCompleteRegistration struct {
	Recovery string `json:"recovery" validate:"required,min=12,max=128"`
}

type RequestAuthLogin struct {
	Passphrase string `json:"passphrase" validate:"required,min=12,max=128"`
}

type ResponseAuthLogin struct {
	Token string `json:"token"`
}

type RequestAuthUpdatePassphrase struct {
	Passphrase string `json:"passphrase" validate:"required,min=12,max=128"`
}

type RequestAuthRecover struct {
	RecoveryKey   string `json:"recoveryKey"`
	NewPassphrase string `json:"newPassphrase"`
}
