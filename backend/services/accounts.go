package services

import (
	"passenger-go/backend/pipes"
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/encrypt"
	"passenger-go/backend/utilities/strength"
	"strconv"

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
	// We need a new repository method that returns encrypted data
	// For now, let's work with the existing approach and get encrypted strength differently
	accounts, err := service.repository.GetAccountsWithEncryptedData()
	if err != nil {
		return nil, err
	}

	// Decrypt each account's fields
	decryptedAccounts := make([]*schemas.ResponseAccount, len(accounts))
	for i, account := range accounts {
		decrypted, err := service.decryptAccountRowToResponse(account)
		if err != nil {
			return nil, err
		}
		decryptedAccounts[i] = decrypted
	}

	return decryptedAccounts, nil
}

func (service *AccountsService) GetAccount(
	id string,
) (*schemas.ResponseAccountDetails, error) {
	// We need a new repository method that returns encrypted data
	account, err := service.repository.GetAccountWithEncryptedData(id)
	if err != nil {
		return nil, err
	}

	// Decrypt the account
	return service.decryptAccountDetailsRowToResponse(account)
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

	// Calculate strength before encryption
	strengthScore, err := strength.CalculateStrength(body.Passphrase)
	if err != nil {
		return nil, err
	}

	// Encrypt all fields
	encryptedBody, err := service.encryptRequestBodyWithStrength(body, strengthScore)
	if err != nil {
		return nil, err
	}

	account, err := service.repository.CreateAccount(encryptedBody)
	if err != nil {
		return nil, err
	}

	// Return decrypted account
	return &schemas.ResponseAccountDetails{
		Id:         account.Id,
		Platform:   body.Platform,
		Identifier: body.Identifier,
		Passphrase: body.Passphrase,
		Url:        body.Url,
		Notes:      body.Notes,
		Strength:   strengthScore,
	}, nil
}

func (service *AccountsService) UpdateAccount(
	id string,
	body *schemas.RequestAccountsUpsert,
) error {
	err := service.validator.Struct(body)
	if err != nil {
		return err
	}

	// Calculate strength before encryption
	strengthScore, err := strength.CalculateStrength(body.Passphrase)
	if err != nil {
		return err
	}

	// Encrypt all fields
	encryptedBody, err := service.encryptRequestBodyWithStrength(body, strengthScore)
	if err != nil {
		return err
	}

	return service.repository.UpdateAccount(id, encryptedBody)
}

func (service *AccountsService) DeleteAccount(
	id string,
) error {
	return service.repository.DeleteAccount(id)
}

func (service *AccountsService) GetUniqueIdentifiers() ([]string, error) {
	encryptedIdentifiers, err := service.repository.GetUniqueIdentifiers()
	if err != nil {
		return nil, err
	}

	// Decrypt all identifiers
	decryptedIdentifiers := make([]string, len(encryptedIdentifiers))
	for i, encryptedId := range encryptedIdentifiers {
		decrypted, err := encrypt.DecryptDeterministic(encryptedId)
		if err != nil {
			return nil, err
		}
		decryptedIdentifiers[i] = decrypted
	}

	return decryptedIdentifiers, nil
}

// Helper function to encrypt request body fields with strength
func (service *AccountsService) encryptRequestBodyWithStrength(body *schemas.RequestAccountsUpsert, strengthScore int) (*schemas.RequestAccountsUpsert, error) {
	encryptedPlatform, err := encrypt.EncryptDeterministic(body.Platform)
	if err != nil {
		return nil, err
	}

	encryptedIdentifier, err := encrypt.EncryptDeterministic(body.Identifier)
	if err != nil {
		return nil, err
	}

	encryptedPassphrase, err := encrypt.Encrypt(body.Passphrase)
	if err != nil {
		return nil, err
	}

	encryptedUrl, err := encrypt.EncryptDeterministic(body.Url)
	if err != nil {
		return nil, err
	}

	encryptedNotes, err := encrypt.EncryptDeterministic(body.Notes)
	if err != nil {
		return nil, err
	}

	// Encrypt strength as string
	encryptedStrength, err := encrypt.EncryptDeterministic(strconv.Itoa(strengthScore))
	if err != nil {
		return nil, err
	}

	return &schemas.RequestAccountsUpsert{
		Platform:   encryptedPlatform,
		Identifier: encryptedIdentifier,
		Passphrase: encryptedPassphrase,
		Url:        encryptedUrl,
		Notes:      encryptedNotes,
		Strength:   encryptedStrength,
	}, nil
}

// Helper function to decrypt account row data
func (service *AccountsService) decryptAccountRowToResponse(account *repositories.EncryptedAccountRow) (*schemas.ResponseAccount, error) {
	decryptedPlatform, err := encrypt.DecryptDeterministic(account.Platform)
	if err != nil {
		return nil, err
	}

	decryptedIdentifier, err := encrypt.DecryptDeterministic(account.Identifier)
	if err != nil {
		return nil, err
	}

	decryptedUrl, err := encrypt.DecryptDeterministic(account.Url)
	if err != nil {
		return nil, err
	}

	decryptedNotes, err := encrypt.DecryptDeterministic(account.Notes)
	if err != nil {
		return nil, err
	}

	// Decrypt and convert strength from string to int
	decryptedStrengthStr, err := encrypt.DecryptDeterministic(account.EncryptedStrength)
	if err != nil {
		return nil, err
	}

	strengthScore, err := strconv.Atoi(decryptedStrengthStr)
	if err != nil {
		return nil, err
	}

	return &schemas.ResponseAccount{
		Id:         account.Id,
		Platform:   decryptedPlatform,
		Identifier: decryptedIdentifier,
		Url:        decryptedUrl,
		Notes:      decryptedNotes,
		Strength:   strengthScore,
	}, nil
}

// Helper function to decrypt account details row data
func (service *AccountsService) decryptAccountDetailsRowToResponse(account *repositories.EncryptedAccountDetailsRow) (*schemas.ResponseAccountDetails, error) {
	decryptedPlatform, err := encrypt.DecryptDeterministic(account.Platform)
	if err != nil {
		return nil, err
	}

	decryptedIdentifier, err := encrypt.DecryptDeterministic(account.Identifier)
	if err != nil {
		return nil, err
	}

	decryptedPassphrase, err := encrypt.Decrypt(account.Passphrase)
	if err != nil {
		return nil, err
	}

	decryptedUrl, err := encrypt.DecryptDeterministic(account.Url)
	if err != nil {
		return nil, err
	}

	decryptedNotes, err := encrypt.DecryptDeterministic(account.Notes)
	if err != nil {
		return nil, err
	}

	// Decrypt and convert strength from string to int
	decryptedStrengthStr, err := encrypt.DecryptDeterministic(account.EncryptedStrength)
	if err != nil {
		return nil, err
	}

	strengthScore, err := strconv.Atoi(decryptedStrengthStr)
	if err != nil {
		return nil, err
	}

	return &schemas.ResponseAccountDetails{
		Id:         account.Id,
		Platform:   decryptedPlatform,
		Identifier: decryptedIdentifier,
		Passphrase: decryptedPassphrase,
		Url:        decryptedUrl,
		Notes:      decryptedNotes,
		Strength:   strengthScore,
	}, nil
}
