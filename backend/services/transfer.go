package services

import (
	"passenger-go/backend/repositories"
)

type TransferService struct {
	repository *repositories.AccountsRepository
}

func NewTransferService() *TransferService {
	return &TransferService{repository: repositories.NewAccountsRepository()}
}

func (service *TransferService) Import() error {
	return nil
}

func (service *TransferService) Export() error {
	return nil
}
