package repositories

import (
	"database/sql"
	"fmt"
	"passenger-go/backend/models"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities"
	"strings"
)

type AccountsRepository struct {
	database *sql.DB
}

func NewAccountsRepository() *AccountsRepository {
	return &AccountsRepository{database: utilities.GetDB()}
}

func (repository *AccountsRepository) CreateAccount(
	account *models.Account,
) (string, *schemas.APIError) {
	statement, err := repository.database.Prepare(QueryAccountCreate)
	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to prepare account create statement",
			err,
		)
	}

	defer statement.Close()

	result, err := statement.Exec(
		account.Platform,
		account.Identifier,
		account.Passphrase,
		account.Notes,
		account.Favorite,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return "", schemas.NewAPIError(
				schemas.ErrAccountAlreadyExists,
				"account already exists",
				err,
			)
		}
		return "", schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to create account",
			err,
		)
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to get last insert id",
			err,
		)
	}

	return fmt.Sprintf("%d", lastInsertId), nil
}

func (repository *AccountsRepository) GetAccountCards(
	page int,
	take int,
) ([]*models.Account, *schemas.APIError) {
	if page < 1 {
		page = 1
	}
	if take < 1 {
		take = 10
	}

	rows, err := repository.database.Query(
		QueryAccountCards,
		take,
		(page-1)*take,
	)
	if err != nil {
		return nil, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to get account cards",
			err,
		)
	}

	defer rows.Close()

	accounts := make([]*models.Account, 0)
	for rows.Next() {
		account := &models.Account{}
		err = rows.Scan(
			&account.Id,
			&account.Platform,
			&account.Identifier,
			&account.Favorite,
		)
		if err != nil {
			return nil, schemas.NewAPIError(
				schemas.ErrDatabase,
				"failed to scan account card",
				err,
			)
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (repository *AccountsRepository) GetAccountDetails(
	id string,
) (*models.Account, *schemas.APIError) {
	row := repository.database.QueryRow(QueryAccountDetails, id)

	account := &models.Account{}
	err := row.Scan(
		&account.Id,
		&account.Platform,
		&account.Identifier,
		&account.Passphrase,
		&account.Notes,
		&account.Favorite,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.DeletedAt,
		&account.AccessCount,
		&account.Strength,
		&account.LastAccessed,
	)
	if err != nil {
		return nil, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to scan account details",
			err,
		)
	}

	return account, nil
}

func (repository *AccountsRepository) UpdateAccount(
	account *models.Account,
) (*models.Account, *schemas.APIError) {
	statement, err := repository.database.Prepare(QueryAccountUpdate)
	if err != nil {
		return nil, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to prepare account update statement",
			err,
		)
	}

	defer statement.Close()

	_, err = statement.Exec(
		account.Platform,
		account.Identifier,
		account.Passphrase,
		account.Notes,
		account.Favorite,
		account.Id,
	)
	if err != nil {
		return nil, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to update account",
			err,
		)
	}

	return account, nil
}

func (repository *AccountsRepository) DeleteAccount(
	id string,
) (*models.Account, *schemas.APIError) {
	statement, err := repository.database.Prepare(QueryAccountDelete)
	if err != nil {
		return nil, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to prepare account delete statement",
			err,
		)
	}
	defer statement.Close()

	_, err = statement.Exec(id)
	if err != nil {
		return nil, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to delete account",
			err,
		)
	}

	return nil, nil
}

func (repository *AccountsRepository) UpdateAccountStrength(
	id string,
	strength int,
) (int, *schemas.APIError) {
	statement, err := repository.database.Prepare(QueryAccountUpdateStrength)
	if err != nil {
		return 0, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to prepare account update strength statement",
			err,
		)
	}
	defer statement.Close()

	_, err = statement.Exec(strength, id)
	if err != nil {
		return 0, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to update account strength",
			err,
		)
	}

	return strength, nil
}

func (repository *AccountsRepository) UpdateAccountAccessed(
	id string,
) *schemas.APIError {
	statement, err := repository.database.Prepare(QueryAccountAccessed)
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to prepare account update accessed statement",
			err,
		)
	}
	defer statement.Close()

	_, err = statement.Exec(id)
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to update account accessed",
			err,
		)
	}

	return nil
}

func (repository *AccountsRepository) GetAccountPassphrase(
	id string,
) (string, *schemas.APIError) {
	row := repository.database.QueryRow(QueryAccountPassphrase, id)

	passphrase := ""
	err := row.Scan(&passphrase)
	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to scan account passphrase",
			err,
		)
	}

	return passphrase, nil
}
