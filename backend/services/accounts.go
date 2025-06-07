package services

import (
	"passenger-go/backend/models"
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
)

type AccountsService struct {
	repository *repositories.AccountsRepository
}

func NewAccountsService() *AccountsService {
	return &AccountsService{
		repository: repositories.NewAccountsRepository(),
	}
}

func (service *AccountsService) CreateAccount(
	account *schemas.RequestAccountsCreate,
) (string, *schemas.APIError) {
	accountModel := &models.Account{
		Platform:   account.Platform,
		Identifier: account.Identifier,
		Passphrase: account.Passphrase,
		Notes:      account.Notes,
		Favorite:   account.Favorite,
	}

	createdAccountId, err := service.repository.CreateAccount(accountModel)
	if err != nil {
		return "", err
	}

	return createdAccountId, nil
}

func (service *AccountsService) GetAccountCards(
	page int,
	take int,
) ([]*models.Account, *schemas.APIError) {
	return service.repository.GetAccountCards(page, take)
}

func (service *AccountsService) GetAccountDetails(
	id string,
) (*models.Account, *schemas.APIError) {
	return service.repository.GetAccountDetails(id)
}

func (service *AccountsService) UpdateAccount(
	id string,
	account *schemas.RequestAccountsUpdate,
) error {
	accountDetails, err := service.repository.GetAccountDetails(id)
	if err != nil {
		return err
	}

	accountModel := &models.Account{
		Id:         accountDetails.Id,
		Platform:   accountDetails.Platform,
		Identifier: accountDetails.Identifier,
		Notes:      account.Notes,
		Favorite:   account.Favorite,
	}

	_, err = service.repository.UpdateAccount(accountModel)
	if err != nil {
		return err
	}

	return nil
}

func (service *AccountsService) DeleteAccount(
	id string,
) error {
	_, err := service.repository.DeleteAccount(id)
	if err != nil {
		return err
	}

	return nil
}
