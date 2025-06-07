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
) (string, error) {
	return service.repository.CreateAccount(&models.Account{
		Platform:   account.Platform,
		Identifier: account.Identifier,
		Passphrase: account.Passphrase,
		Notes:      account.Notes,
		Favorite:   account.Favorite,
	})
}

func (service *AccountsService) GetAccountCards(
	page int,
	take int,
) ([]*schemas.ResponseAccountCard, error) {
	return service.repository.GetAccountCards(page, take)
}

func (service *AccountsService) GetAccountDetails(
	id string,
) (*schemas.ResponseAccountDetails, error) {
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

	return service.repository.UpdateAccount(accountModel)
}

func (service *AccountsService) DeleteAccount(
	id string,
) error {
	return service.repository.DeleteAccount(id)
}
