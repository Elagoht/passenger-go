package services

import (
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
	return service.repository.CreateAccount(&schemas.RequestAccountsCreate{
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
	return service.repository.UpdateAccount(&schemas.RequestAccountsCreate{
		Platform:   account.Platform,
		Identifier: account.Identifier,
		Passphrase: account.Passphrase,
		Url:        account.Url,
		Notes:      account.Notes,
		Favorite:   account.Favorite,
	})
}

func (service *AccountsService) DeleteAccount(
	id string,
) error {
	return service.repository.DeleteAccount(id)
}
