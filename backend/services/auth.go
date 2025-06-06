/**
 * This is a single user self-hosted password manager.
 * The user can register, login, and update their passphrase.
 * Service can create a user but cannot delete it.
 * JWT based stateless authentication is used.
 *
 * Register Flow:
 * 1. User requests to register
 * 2. Service creates a temporary user
 * 3. Service returns a recovery key
 * 4. User will asked for this recovery key
 *    - If the recovery key is correct, the temporary user will be converted to a permanent user
 *    - If the recovery key is incorrect, the temporary user will be deleted
 *
 * If the user tries to register again without the completion of the registration,
 * the temporary user will be deleted.
 */

package services

import (
	"passenger-go/backend/pipes"
	"passenger-go/backend/repositories"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities"

	"github.com/go-playground/validator/v10"
)

type AuthService struct {
	repository *repositories.AuthRepository
	validate   *validator.Validate
}

func NewAuthService() *AuthService {
	return &AuthService{
		repository: repositories.NewAuthRepository(),
		validate:   pipes.GetValidator(),
	}
}

// Check if the application is initialized by creating a user
func (service *AuthService) Status() (bool, error) {
	count, err := service.repository.GetUserCount()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (service *AuthService) RegisterUser(passphrase string) (string, error) {
	initialized, err := service.Status()
	if err != nil {
		return "", err
	}

	if initialized {
		return "", schemas.NewAPIError(
			schemas.ErrAlreadyInitialized,
			"You have already initialized the application",
			nil,
		)
	}

	encryptedPassphrase, err := utilities.Encrypt(passphrase)
	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrEncryptionFailed,
			"Couldn't encrypt passphrase",
			err,
		)
	}

	recoveryKey, err := utilities.GenerateRecoveryKey(passphrase)

	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrRecoveryGenerationFailed,
			"Failed to generate recovery key",
			err,
		)
	}

	// Create a temporary user
	err = service.repository.CreateUser(encryptedPassphrase, recoveryKey)
	if err != nil {
		return "", err
	}

	return recoveryKey, nil
}

func (service *AuthService) CompleteRegistration(recovery string) error {
	user, err := service.repository.GetUser()
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrNotInitializedYet,
			"You haven't initialized the application yet",
			err,
		)
	}

	if user.Validated {
		return schemas.NewAPIError(
			schemas.ErrAlreadyInitialized,
			"You have already completed the registration",
			nil,
		)
	}

	if user.Recovery != recovery {
		return schemas.NewAPIError(
			schemas.ErrInvalidCredentials,
			"Invalid recovery key",
			nil,
		)
	}

	return service.repository.ValidateUser()
}

// Generate a JWT token for the user
func (service *AuthService) LoginUser(passphrase string) (string, error) {
	user, err := service.repository.GetUser()
	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrNotInitializedYet,
			"You haven't initialized the application yet",
			err,
		)
	}

	encryptedPassphrase, err := utilities.Encrypt(passphrase)
	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrEncryptionFailed,
			"Couldn't encrypt passphrase",
			err,
		)
	}

	if user.Passphrase != encryptedPassphrase {
		return "", schemas.NewAPIError(
			schemas.ErrInvalidCredentials,
			"Invalid credentials",
			nil,
		)
	}

	token, err := utilities.GenerateJWT(user.Id)
	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrJWTGenerationFailed,
			"Failed to generate JWT",
			err,
		)
	}

	return token, nil
}

// Protected by JWT token
func (service *AuthService) UpdatePassphrase(newPassphrase string) error {
	initialized, err := service.Status()
	if err != nil {
		return err
	}

	if !initialized {
		return schemas.NewAPIError(
			schemas.ErrNotInitializedYet,
			"You haven't initialized the application yet",
			nil,
		)
	}

	encryptedNewPassphrase, err := utilities.Encrypt(newPassphrase)
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrEncryptionFailed,
			"Couldn't encrypt passphrase",
			err,
		)
	}

	return service.repository.UpdateUser(encryptedNewPassphrase)
}
