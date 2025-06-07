package repositories

import (
	"database/sql"
	"passenger-go/backend/models"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities"
)

type AuthRepository struct {
	database *sql.DB
}

func NewAuthRepository() *AuthRepository {
	return &AuthRepository{database: utilities.GetDB()}
}

func (repository *AuthRepository) GetUserCount() (int, *schemas.APIError) {
	rows, err := repository.database.Query(QueryGetUserCount)
	if err != nil {
		return 0, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to get user count",
			err,
		)
	}

	defer rows.Close()

	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return 0, schemas.NewAPIError(
				schemas.ErrDatabase,
				"failed to get user count",
				err,
			)
		}
	}

	return count, nil
}

func (repository *AuthRepository) GetUser() (*models.User, error) {
	row := repository.database.QueryRow(QueryGetUser)

	var user models.User
	err := row.Scan(&user.Id, &user.Passphrase, &user.Validated, &user.Recovery)
	if err != nil {
		return nil, schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to get user",
			err,
		)
	}

	return &user, nil
}

func (repository *AuthRepository) CreateUser(
	passphrase string,
	recoveryKey string,
) error {
	_, err := repository.database.Exec(QueryCreateUser, passphrase, recoveryKey)
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to create user",
			err,
		)
	}

	return nil
}

func (repository *AuthRepository) ValidateUser() error {
	rows, err := repository.database.Query(QueryValidateUser)
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to validate user",
			err,
		)
	}

	var validated bool
	for rows.Next() {
		err := rows.Scan(&validated)
		if err != nil {
			return schemas.NewAPIError(
				schemas.ErrDatabase,
				"failed to validate user",
				err,
			)
		}
	}

	if validated {
		return schemas.NewAPIError(
			schemas.ErrAlreadyInitialized,
			"user already validated",
			nil,
		)
	}

	_, err = repository.database.Exec(QueryValidateUser)
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to validate user",
			err,
		)
	}
	return nil
}

func (repository *AuthRepository) UpdateUser(
	passphrase string,
) error {
	_, err := repository.database.Exec(QueryUpdatePassphrase, passphrase)
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrDatabase,
			"failed to update user",
			err,
		)
	}

	return nil
}
