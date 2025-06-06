package utilities

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

type database struct {
	connection *sql.DB
	mutex      sync.RWMutex
}

var (
	instance *database
	once     sync.Once
)

func init() {
	godotenv.Load()
	initializeTables()
}

// A singleton instance of the database connection
func GetDB() *sql.DB {
	once.Do(func() {
		instance = &database{}
	})
	connection, err := instance.getConnection()
	if err != nil {
		log.Fatal(err)
	}
	return connection
}

func (database *database) connect() error {
	database.mutex.Lock()
	defer database.mutex.Unlock()

	if database.connection != nil { // Already connected
		return nil
	}

	passphrase := os.Getenv("DB_PASSPHRASE")
	if passphrase == "" {
		return fmt.Errorf(
			"DB_PASSPHRASE environment variable is not set",
		)
	}

	connectionString := fmt.Sprintf(
		"file:passenger.db?_pragma_key=%s&_pragma_cipher_page_size=4096",
		passphrase,
	)

	connection, err := sql.Open("sqlite", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize the database with encryption
	if _, err := connection.Exec(
		"PRAGMA cipher_compatibility = 4",
	); err != nil {
		connection.Close()
		return fmt.Errorf("failed to set cipher compatibility: %w", err)
	}

	// Verify the connection
	if err := connection.Ping(); err != nil {
		connection.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	database.connection = connection
	return nil
}

func (database *database) getConnection() (*sql.DB, error) {
	database.mutex.RLock()
	if database.connection != nil {
		defer database.mutex.RUnlock()
		return database.connection, nil
	}
	database.mutex.RUnlock()

	if err := database.connect(); err != nil {
		return nil, err
	}

	database.mutex.RLock()
	defer database.mutex.RUnlock()
	return database.connection, nil
}

// Use this when the application is shutting down
func (database *database) Close() error {
	database.mutex.Lock()
	defer database.mutex.Unlock()

	if database.connection != nil {
		err := database.connection.Close()
		database.connection = nil
		return err
	}
	return nil
}

func initializeTables() {
	database := GetDB()

	queries := []string{
		QueryCreateUserTable,
		QueryCreateAccountsTable,
		QuerySeedUser,
	}

	for _, query := range queries {
		_, err := database.Exec(query)
		if err != nil {
			panic(err)
		}
	}
}
