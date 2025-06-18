package services

import (
	"passenger-go/backend/pipes"
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/encrypt"
	"passenger-go/backend/utilities/strength"

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

	decryptedAccounts := []*schemas.ResponseAccount{}

	for _, account := range accounts {
		decryptedData, err := generateDecryptedListData(account)
		if err != nil {
			return nil, err
		}

		decryptedAccounts = append(decryptedAccounts, decryptedData)
	}

	return decryptedAccounts, nil
}

func (service *AccountsService) GetAccount(
	id string,
) (*schemas.ResponseAccountDetails, error) {
	account, err := service.repository.GetAccount(id)
	if err != nil {
		return nil, err
	}

	decryptedData, err := generateDectyptedData(account)
	if err != nil {
		return nil, err
	}

	return decryptedData, nil
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
	err := service.validator.Struct(body)
	if err != nil {
		return nil, err
	}

	strengthScore, err := strength.CalculateStrength(body.Passphrase)
	if err != nil {
		return nil, err
	}

	encryptedData, err := generateEncryptedData(body)
	if err != nil {
		return nil, err
	}

	return service.repository.CreateAccount(encryptedData, strengthScore)
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

func generateEncryptedData(body *schemas.RequestAccountsUpsert) (*schemas.RequestAccountsUpsert, error) {
	encryptedPlatform, err := encrypt.Encrypt(body.Platform)
	if err != nil {
		return nil, err
	}

	encryptedIdentifier, err := encrypt.Encrypt(body.Identifier)
	if err != nil {
		return nil, err
	}

	encryptedUrl, err := encrypt.Encrypt(body.Url)
	if err != nil {
		return nil, err
	}

	encryptedPassphrase, err := encrypt.Encrypt(body.Passphrase)
	if err != nil {
		return nil, err
	}

	encryptedNotes, err := encrypt.Encrypt(body.Notes)
	if err != nil {
		return nil, err
	}

	return &schemas.RequestAccountsUpsert{
		Platform:   encryptedPlatform,
		Identifier: encryptedIdentifier,
		Passphrase: encryptedPassphrase,
		Url:        encryptedUrl,
		Notes:      encryptedNotes,
	}, nil
}

func generateDectyptedData(body *schemas.ResponseAccountDetails) (*schemas.ResponseAccountDetails, error) {
	decryptedPlatform, err := encrypt.Decrypt(body.Platform)
	if err != nil {
		return nil, err
	}

	decryptedIdentifier, err := encrypt.Decrypt(body.Identifier)
	if err != nil {
		return nil, err
	}

	decryptedPassphrase, err := encrypt.Decrypt(body.Passphrase)
	if err != nil {
		return nil, err
	}

	decryptedUrl, err := encrypt.Decrypt(body.Url)
	if err != nil {
		return nil, err
	}

	decryptedNotes, err := encrypt.Decrypt(body.Notes)
	if err != nil {
		return nil, err
	}

	return &schemas.ResponseAccountDetails{
		Id:         body.Id,
		Platform:   decryptedPlatform,
		Identifier: decryptedIdentifier,
		Url:        decryptedUrl,
		Passphrase: decryptedPassphrase,
		Notes:      decryptedNotes,
		Strength:   body.Strength,
	}, nil
}

func generateDecryptedListData(body *schemas.ResponseAccount) (*schemas.ResponseAccount, error) {
	decryptedPlatform, err := encrypt.Decrypt(body.Platform)
	if err != nil {
		return nil, err
	}

	decryptedIdentifier, err := encrypt.Decrypt(body.Identifier)
	if err != nil {
		return nil, err
	}

	decryptedUrl, err := encrypt.Decrypt(body.Url)
	if err != nil {
		return nil, err
	}

	decryptedNotes, err := encrypt.Decrypt(body.Notes)
	if err != nil {
		return nil, err
	}

	return &schemas.ResponseAccount{
		Id:         body.Id,
		Platform:   decryptedPlatform,
		Identifier: decryptedIdentifier,
		Url:        decryptedUrl,
		Notes:      decryptedNotes,
		Strength:   body.Strength,
	}, nil
}
