package services

import (
	"fmt"
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/encrypt"
)

type TransferService struct {
	accountsService *AccountsService
	repository      *repositories.AccountsRepository
}

func NewTransferService() *TransferService {
	return &TransferService{
		accountsService: NewAccountsService(),
		repository:      repositories.NewAccountsRepository(),
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
		_, err := service.accountsService.CreateAccount(&account)
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

func (service *TransferService) Export() (string, error) {
	accounts, err := service.repository.ExportAccountsData()
	if err != nil {
		return "", err
	}

	csv := "platform,identifier,passphrase,url,notes\n"
	for _, account := range accounts {
		decryptedPassphrase, err := encrypt.Decrypt(account.Passphrase)
		if err != nil {
			return "", err
		}
		account.Passphrase = decryptedPassphrase
		csv += convertAccountToCSV(account)
	}

	return csv, nil
}

func convertAccountToCSV(account schemas.RequestAccountsUpsert) string {
	return fmt.Sprintf("%s,%s,%s,%s,%s",
		account.Platform,
		account.Identifier,
		account.Passphrase,
		account.Url,
		account.Notes,
	)
}
