package services

import (
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
)

type TransferService struct {
	repository *repositories.AccountsRepository
}

func NewTransferService() *TransferService {
	return &TransferService{
		repository: repositories.NewAccountsRepository(),
	}
}

type ImportResult struct {
	SuccessCount int                             `json:"successCount"`
	FailedOnes   []schemas.RequestAccountsUpsert `json:"failedOnes"`
}

func (service *TransferService) Import(
	accounts []schemas.RequestAccountsUpsert,
) (*ImportResult, error) {
	successCount := 0
	failedOnes := []schemas.RequestAccountsUpsert{}

	for _, account := range accounts {
		_, err := service.repository.CreateAccount(&schemas.RequestAccountsUpsert{
			Platform:   account.Platform,
			Identifier: account.Identifier,
			Passphrase: account.Passphrase,
			Url:        account.Url,
			Notes:      account.Notes,
		})
		if err != nil {
			failedOnes = append(failedOnes, account)
			continue
		}
		successCount++
	}

	if successCount > 0 {
		return &ImportResult{
			SuccessCount: successCount,
			FailedOnes:   failedOnes,
		}, nil
	}

	return &ImportResult{
		SuccessCount: successCount,
		FailedOnes:   failedOnes,
	}, nil
}

func (service *TransferService) Export() error {
	return nil
}
