package repositories

import (
	"database/sql"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/database"
	"strconv"
	"strings"
)

type AccountsRepository struct {
	database *sql.DB
}

func NewAccountsRepository() *AccountsRepository {
	return &AccountsRepository{database: database.GetDB()}
}

// Exported structs for encrypted data
type EncryptedAccountRow struct {
	Id                string
	Platform          string
	Identifier        string
	Url               string
	Notes             string
	EncryptedStrength string
}

type EncryptedAccountDetailsRow struct {
	Id                string
	Platform          string
	Identifier        string
	Url               string
	Passphrase        string
	Notes             string
	EncryptedStrength string
}

func (repository *AccountsRepository) GetAccounts() ([]*schemas.ResponseAccount, error) {
	statement, err := repository.database.Prepare(QueryAccounts)
	if err != nil {
		return nil, err
	}

	rows, err := statement.Query()
	if err != nil {
		return nil, err
	}

	accounts := []*schemas.ResponseAccount{}

	for rows.Next() {
		var row EncryptedAccountRow
		err = rows.Scan(
			&row.Id,
			&row.Platform,
			&row.Identifier,
			&row.Url,
			&row.Notes,
			&row.EncryptedStrength,
		)
		if err != nil {
			return nil, err
		}

		// Convert to ResponseAccount with encrypted data
		// The service layer will decrypt these
		account := &schemas.ResponseAccount{
			Id:         row.Id,
			Platform:   row.Platform,
			Identifier: row.Identifier,
			Url:        row.Url,
			Notes:      row.Notes,
			Strength:   0, // Will be set after decryption by service
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (repository *AccountsRepository) GetAccountsWithEncryptedData() ([]*EncryptedAccountRow, error) {
	statement, err := repository.database.Prepare(QueryAccounts)
	if err != nil {
		return nil, err
	}

	rows, err := statement.Query()
	if err != nil {
		return nil, err
	}

	accounts := []*EncryptedAccountRow{}

	for rows.Next() {
		var row EncryptedAccountRow
		err = rows.Scan(
			&row.Id,
			&row.Platform,
			&row.Identifier,
			&row.Url,
			&row.Notes,
			&row.EncryptedStrength,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, &row)
	}

	return accounts, nil
}

func (repository *AccountsRepository) GetAccount(
	id string,
) (*schemas.ResponseAccountDetails, error) {
	statement, err := repository.database.Prepare(QueryAccountDetails)
	if err != nil {
		return nil, err
	}

	var row EncryptedAccountDetailsRow
	err = statement.QueryRow(id).Scan(
		&row.Id,
		&row.Platform,
		&row.Identifier,
		&row.Url,
		&row.Passphrase,
		&row.Notes,
		&row.EncryptedStrength,
	)
	if err != nil {
		return nil, err
	}

	// Return with encrypted data - service layer will decrypt
	return &schemas.ResponseAccountDetails{
		Id:         row.Id,
		Platform:   row.Platform,
		Identifier: row.Identifier,
		Url:        row.Url,
		Passphrase: row.Passphrase,
		Notes:      row.Notes,
		Strength:   0, // Will be set after decryption by service
	}, nil
}

func (repository *AccountsRepository) GetAccountWithEncryptedData(
	id string,
) (*EncryptedAccountDetailsRow, error) {
	statement, err := repository.database.Prepare(QueryAccountDetails)
	if err != nil {
		return nil, err
	}

	var row EncryptedAccountDetailsRow
	err = statement.QueryRow(id).Scan(
		&row.Id,
		&row.Platform,
		&row.Identifier,
		&row.Url,
		&row.Passphrase,
		&row.Notes,
		&row.EncryptedStrength,
	)
	if err != nil {
		return nil, err
	}

	return &row, nil
}

func (repository *AccountsRepository) GetPassphrase(
	id string,
) (string, error) {
	statement, err := repository.database.Prepare(QueryAccountPassphrase)
	if err != nil {
		return "", err
	}

	var passphrase string
	err = statement.QueryRow(id).Scan(&passphrase)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", schemas.NewAPIError(
				schemas.ErrAccountNotFound,
				"Account not found",
				nil,
			)
		}
		return "", err
	}

	return passphrase, nil
}

func (repository *AccountsRepository) CreateAccount(
	account *schemas.RequestAccountsUpsert,
) (*schemas.ResponseAccountDetails, error) {
	statement, err := repository.database.Prepare(QueryAccountCreate)
	if err != nil {
		return nil, err
	}

	result, err := statement.Exec(
		account.Platform,
		account.Identifier,
		account.Passphrase,
		account.Url,
		account.Notes,
		account.Strength, // This is the encrypted strength from service
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, schemas.NewAPIError(
				schemas.ErrAccountAlreadyExists,
				"Account already exists",
				nil,
			)
		}
		return nil, err
	}

	lastInsertedId, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &schemas.ResponseAccountDetails{
		Id:         strconv.FormatInt(lastInsertedId, 10),
		Platform:   account.Platform,
		Identifier: account.Identifier,
		Passphrase: account.Passphrase,
		Url:        account.Url,
		Notes:      account.Notes,
		Strength:   0, // Will be set by service layer after decryption
	}, nil
}

func (repository *AccountsRepository) UpdateAccount(
	id string,
	account *schemas.RequestAccountsUpsert,
) error {
	statement, err := repository.database.Prepare(QueryAccountUpdate)
	if err != nil {
		return err
	}

	_, err = statement.Exec(
		account.Platform,
		account.Identifier,
		account.Passphrase,
		account.Url,
		account.Notes,
		account.Strength, // This is the encrypted strength from service
		id,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return schemas.NewAPIError(
				schemas.ErrAnotherAccountFound,
				"An account with the same platform and identifier already exists",
				nil,
			)
		}
		return err
	}

	return nil
}

func (repository *AccountsRepository) DeleteAccount(
	id string,
) error {
	statement, err := repository.database.Prepare(QueryAccountDelete)
	if err != nil {
		return err
	}

	result, err := statement.Exec(id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return schemas.NewAPIError(
			schemas.ErrAccountNotFound,
			"Account not found",
			nil,
		)
	}

	return nil
}

func (repository *AccountsRepository) ExportAccountsData() ([]schemas.RequestAccountsUpsert, error) {
	statement, err := repository.database.Prepare(QueryAccountsExport)
	if err != nil {
		return nil, err
	}

	rows, err := statement.Query()
	if err != nil {
		return nil, err
	}

	accounts := []schemas.RequestAccountsUpsert{}

	for rows.Next() {
		var account schemas.RequestAccountsUpsert
		err = rows.Scan(
			&account.Platform,
			&account.Identifier,
			&account.Passphrase,
			&account.Url,
			&account.Notes,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (repository *AccountsRepository) GetUniqueIdentifiers() ([]string, error) {
	statement, err := repository.database.Prepare(QueryUniqueIdentifiers)
	if err != nil {
		return nil, err
	}

	rows, err := statement.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identifiers []string
	for rows.Next() {
		var identifier string
		err = rows.Scan(&identifier)
		if err != nil {
			return nil, err
		}
		identifiers = append(identifiers, identifier)
	}

	return identifiers, nil
}
