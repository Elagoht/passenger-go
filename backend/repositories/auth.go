package repositories

import (
	"database/sql"
	"errors"
	"passenger-go/backend/models"
	"passenger-go/backend/utilities"
)

type AuthRepository struct {
	database *sql.DB
}

func NewAuthRepository() *AuthRepository {
	return &AuthRepository{database: utilities.GetDB()}
}

func (repository *AuthRepository) GetUserCount() (int, error) {
	rows, err := repository.database.Query(QueryGetUserCount)
	if err != nil {
		return 0, err
	}

	defer rows.Close()

	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}

func (repository *AuthRepository) GetUser() (*models.User, error) {
	row := repository.database.QueryRow(QueryGetUser)

	var user models.User
	err := row.Scan(&user.Id, &user.Passphrase, &user.Validated, &user.Recovery)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (repository *AuthRepository) CreateUser(passphrase string, recoveryKey string) error {
	_, err := repository.database.Exec(QueryCreateUser, passphrase, recoveryKey)
	if err != nil {
		return err
	}

	return nil
}

func (repository *AuthRepository) ValidateUser() error {
	rows, err := repository.database.Query(QueryValidateUser)
	if err != nil {
		return err
	}

	var validated bool
	for rows.Next() {
		err := rows.Scan(&validated)
		if err != nil {
			return err
		}
	}

	if validated {
		return errors.New("ALREADY_VALIDATED")
	}

	_, err = repository.database.Exec(QueryValidateUser)
	if err != nil {
		return errors.New("failed to validate user")
	}
	return nil
}

func (repository *AuthRepository) UpdateUser(passphrase string) error {
	_, err := repository.database.Exec(QueryUpdatePassphrase, passphrase)
	if err != nil {
		return err
	}

	return nil
}
