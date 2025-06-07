package services

import (
	"passenger-go/backend/models"
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
)

type TransferService struct {
	repository *repositories.AccountsRepository
}

func NewTransferService() *TransferService {
	return &TransferService{repository: repositories.NewAccountsRepository()}
}

type ImportResult struct {
	SuccessCount int
	FailedOnes   []schemas.RequestAccountsCreate
}

func (service *TransferService) Import(
	accounts []schemas.RequestAccountsCreate,
) (*ImportResult, *schemas.APIError) {
	successCount := 0
	failedOnes := []schemas.RequestAccountsCreate{}

	for _, account := range accounts {
		_, err := service.repository.CreateAccount(&models.Account{
			Platform:   account.Platform,
			Identifier: account.Identifier,
			Passphrase: account.Passphrase,
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
