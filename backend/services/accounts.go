package services

import (
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/encrypt"
)

type AccountsService struct {
	repository *repositories.AccountsRepository
}

func NewAccountsService() *AccountsService {
	return &AccountsService{
		repository: repositories.NewAccountsRepository(),
	}
}

func (service *AccountsService) GetAccounts() ([]*schemas.ResponseAccount, error) {
	accounts, err := service.repository.GetAccounts()
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (service *AccountsService) GetPassphrase(id string) (string, error) {
	passphrase, err := service.repository.GetPassphrase(id)
	if err != nil {
		return "", err
	}

	decryptedPassphrase, err := encrypt.Decrypt(passphrase)
	if err != nil {
		return "", err
	}

	return decryptedPassphrase, nil
}

func (service *AccountsService) CreateAccount(
	body *schemas.RequestAccountsUpsert,
) (*schemas.ResponseAccountsCreate, error) {
	encryptedPassphrase, err := encrypt.Encrypt(body.Passphrase)
	if err != nil {
		return nil, err
	}

	account, err := service.repository.CreateAccount(&schemas.RequestAccountsUpsert{
		Platform:   body.Platform,
		Identifier: body.Identifier,
		Passphrase: encryptedPassphrase,
		Url:        body.Url,
		Notes:      body.Notes,
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (service *AccountsService) UpdateAccount(
	id string,
	body *schemas.RequestAccountsUpsert,
) error {
	encryptedPassphrase, err := encrypt.Encrypt(body.Passphrase)
	if err != nil {
		return err
	}

	err = service.repository.UpdateAccount(id, &schemas.RequestAccountsUpsert{
		Platform:   body.Platform,
		Identifier: body.Identifier,
		Passphrase: encryptedPassphrase,
		Url:        body.Url,
		Notes:      body.Notes,
	})

	if err != nil {
		return err
	}

	return nil
}

func (service *AccountsService) DeleteAccount(id string) error {
	err := service.repository.DeleteAccount(id)
	if err != nil {
		return err
	}

	return nil
}
