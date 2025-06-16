package services

import (
	"passenger-go/backend/pipes"
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/encrypt"

	"github.com/go-playground/validator/v10"
)

type AccountsService struct {
	repository *repositories.AccountsRepository
	validator  *validator.Validate
}

func NewAccountsService() *AccountsService {
	return &AccountsService{
		repository: repositories.NewAccountsRepository(),
		validator:  pipes.GetValidator(),
	}
}

func (service *AccountsService) GetAccounts() ([]*schemas.ResponseAccount, error) {
	accounts, err := service.repository.GetAccounts()
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (service *AccountsService) GetAccount(
	id string,
) (*schemas.ResponseAccountDetails, error) {
	account, err := service.repository.GetAccount(id)
	if err != nil {
		return nil, err
	}

	decryptedPassphrase, err := service.GetPassphrase(id)
	if err != nil {
		return nil, err
	}
	account.Passphrase = decryptedPassphrase

	return account, nil
}

func (service *AccountsService) GetPassphrase(
	id string,
) (string, error) {
	passphrase, err := service.repository.GetPassphrase(id)
	if err != nil {
		return "", err
	}

	return encrypt.Decrypt(passphrase)
}

func (service *AccountsService) CreateAccount(
	body *schemas.RequestAccountsUpsert,
) (*schemas.ResponseAccountDetails, error) {
	encryptedPassphrase, err := encrypt.Encrypt(body.Passphrase)
	if err != nil {
		return nil, err
	}

	err = service.validator.Struct(body)
	if err != nil {
		return nil, err
	}

	return service.repository.CreateAccount(&schemas.ResponseAccountDetails{
		Platform:   body.Platform,
		Identifier: body.Identifier,
		Passphrase: encryptedPassphrase,
		Url:        body.Url,
		Notes:      body.Notes,
	}, body.Passphrase)
}

func (service *AccountsService) UpdateAccount(
	id string,
	body *schemas.RequestAccountsUpsert,
) error {
	encryptedPassphrase, err := encrypt.Encrypt(body.Passphrase)
	if err != nil {
		return err
	}

	err = service.validator.Struct(body)
	if err != nil {
		return err
	}

	return service.repository.UpdateAccount(id, &schemas.RequestAccountsUpsert{
		Platform:   body.Platform,
		Identifier: body.Identifier,
		Passphrase: encryptedPassphrase,
		Url:        body.Url,
		Notes:      body.Notes,
	})
}

func (service *AccountsService) DeleteAccount(
	id string,
) error {
	return service.repository.DeleteAccount(id)
}
