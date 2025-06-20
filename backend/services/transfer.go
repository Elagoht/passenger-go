package services

import (
	"fmt"
	"passenger-go/backend/schemas"
)

type TransferService struct {
	accountsService *AccountsService
}

func NewTransferService() *TransferService {
	return &TransferService{
		accountsService: NewAccountsService(),
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
		_, err := service.accountsService.CreateAccount(&schemas.RequestAccountsUpsert{
			Platform:   account.Platform,
			Identifier: account.Identifier,
			Passphrase: account.Passphrase,
			Url:        account.Url,
			Notes:      account.Notes,
			// Strength will be calculated automatically in the service
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

func (service *TransferService) Export() (string, error) {
	accounts, err := service.accountsService.GetAccounts()
	if err != nil {
		return "", err
	}

	csv := "platform,identifier,passphrase,url,notes\n"
	for _, account := range accounts {
		// Get the full account details including the decrypted passphrase
		fullAccount, err := service.accountsService.GetAccount(account.Id)
		if err != nil {
			return "", err
		}

		csv += convertAccountToCSV(schemas.RequestAccountsUpsert{
			Platform:   fullAccount.Platform,
			Identifier: fullAccount.Identifier,
			Passphrase: fullAccount.Passphrase,
			Url:        fullAccount.Url,
			Notes:      fullAccount.Notes,
		})
	}

	return csv, nil
}

func convertAccountToCSV(account schemas.RequestAccountsUpsert) string {
	return fmt.Sprintf("%s,%s,%s,%s,%s\n",
		account.Platform,
		account.Identifier,
		account.Passphrase,
		account.Url,
		account.Notes,
	)
}
