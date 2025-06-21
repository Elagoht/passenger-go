package database

import (
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"
)

func TestGetDB_SingletonPattern(test *testing.T) {
	cleanupTestDB(test)

	db1 := GetDB()
	db2 := GetDB()

	if db1 != db2 {
		test.Error("GetDB should return the same instance (singleton pattern)")
	}

	if err := db1.Ping(); err != nil {
		test.Errorf("Database ping failed: %v", err)
	}
}

func TestGetDB_ConcurrentAccess(test *testing.T) {
	cleanupTestDB(test)

	var waitGroup sync.WaitGroup
	connections := make([]*sql.DB, 10)

	for index := range 10 {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			connections[index] = GetDB()
		}(index)
	}

	waitGroup.Wait()

	firstConnection := connections[0]
	for index, connection := range connections {
		if connection != firstConnection {
			test.Errorf(
				"Connection %d is not the same as the first connection",
				index,
			)
		}
	}

	if err := firstConnection.Ping(); err != nil {
		test.Errorf("Database ping failed: %v", err)
	}
}

func TestDatabase_Connection(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	if err := db.Ping(); err != nil {
		test.Errorf("Database ping failed: %v", err)
	}

	_, err := db.Exec("SELECT 1")
	if err != nil {
		test.Errorf("Failed to execute simple query: %v", err)
	}
}

func TestDatabase_TableInitialization(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	initializeTablesForTest(test, db)

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM user").Scan(&count)
	if err != nil {
		test.Errorf("Failed to query user table: %v", err)
	}

	if count != 1 {
		test.Errorf("Expected 1 user in table, got %d", count)
	}

	_, err = db.Exec("SELECT COUNT(*) FROM accounts")
	if err != nil {
		test.Errorf("Failed to query accounts table: %v", err)
	}
}

func TestDatabase_TableStructure(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	initializeTablesForTest(test, db)

	rows, err := db.Query("PRAGMA table_info(user)")
	if err != nil {
		test.Errorf("Failed to get user table info: %v", err)
	}
	defer rows.Close()

	expectedColumns := map[string]string{
		"id":         "INTEGER",
		"passphrase": "TEXT",
		"recovery":   "TEXT",
		"validated":  "BOOLEAN",
	}

	columnCount := 0
	for rows.Next() {
		var columnId, notNull, primaryKey int
		var name, typeName string
		var dfltValue sql.NullString
		err := rows.Scan(&columnId, &name, &typeName, &notNull, &dfltValue, &primaryKey)
		if err != nil {
			test.Errorf("Failed to scan table info: %v", err)
		}

		if expectedType, exists := expectedColumns[name]; exists {
			if typeName != expectedType {
				test.Errorf(
					"Column %s has type %s, expected %s",
					name,
					typeName,
					expectedType,
				)
			}
			columnCount++
		}
	}

	if columnCount != len(expectedColumns) {
		test.Errorf(
			"Expected %d columns, found %d",
			len(expectedColumns),
			columnCount,
		)
	}

	rows, err = db.Query("PRAGMA table_info(accounts)")
	if err != nil {
		test.Errorf("Failed to get accounts table info: %v", err)
	}
	defer rows.Close()

	expectedAccountColumns := map[string]string{
		"id":         "INTEGER",
		"platform":   "TEXT",
		"identifier": "TEXT",
		"url":        "TEXT",
		"passphrase": "TEXT",
		"notes":      "TEXT",
		"strength":   "TEXT",
	}

	columnCount = 0
	for rows.Next() {
		var columnId, notNull, primaryKey int
		var name, typeName string
		var dfltValue sql.NullString
		err := rows.Scan(
			&columnId,
			&name,
			&typeName,
			&notNull,
			&dfltValue,
			&primaryKey,
		)
		if err != nil {
			test.Errorf("Failed to scan table info: %v", err)
		}

		if expectedType, exists := expectedAccountColumns[name]; exists {
			if typeName != expectedType {
				test.Errorf(
					"Column %s has type %s, expected %s",
					name,
					typeName,
					expectedType,
				)
			}
			columnCount++
		}
	}

	if columnCount != len(expectedAccountColumns) {
		test.Errorf(
			"Expected %d columns, found %d",
			len(expectedAccountColumns),
			columnCount,
		)
	}
}

func TestDatabase_UniqueConstraints(t *testing.T) {
	cleanupTestDB(t)

	db := GetDB()

	initializeTablesForTest(t, db)

	_, err := db.Exec("INSERT INTO user (passphrase, recovery) VALUES ('test1', 'recovery1')")
	if err != nil {
		t.Errorf("Failed to insert first user: %v", err)
	}

	_, err = db.Exec("INSERT INTO user (passphrase, recovery) VALUES ('test1', 'recovery2')")
	if err == nil {
		t.Error("Expected error when inserting duplicate passphrase, got none")
	}

	_, err = db.Exec("INSERT INTO accounts (platform, identifier, passphrase) VALUES ('test', 'user1', 'pass1')")
	if err != nil {
		t.Errorf("Failed to insert first account: %v", err)
	}

	_, err = db.Exec("INSERT INTO accounts (platform, identifier, passphrase) VALUES ('test', 'user1', 'pass2')")
	if err == nil {
		t.Error("Expected error when inserting duplicate platform+identifier, got none")
	}
}

func TestDatabase_SeedUser(t *testing.T) {
	cleanupTestDB(t)

	db := GetDB()

	initializeTablesForTest(t, db)

	var passphrase, recovery string
	err := db.QueryRow(
		"SELECT passphrase, recovery FROM user LIMIT 1",
	).Scan(&passphrase, &recovery)
	if err != nil {
		t.Errorf("Failed to query seed user: %v", err)
	}

	if passphrase != "" {
		t.Errorf("Expected empty passphrase for seed user, got: %s", passphrase)
	}
	if recovery != "" {
		t.Errorf("Expected empty recovery for seed user, got: %s", recovery)
	}

	_, err = db.Exec(QuerySeedUser)
	if err != nil {
		t.Errorf("Failed to execute seed user query: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user").Scan(&count)
	if err != nil {
		t.Errorf("Failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user after seeding, got %d", count)
	}
}

func TestDatabase_ConcurrentOperations(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := db.Ping(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		test.Errorf("Concurrent ping failed: %v", err)
	}
}

func TestDatabase_TransactionSupport(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	initializeTablesForTest(test, db)

	tx, err := db.Begin()
	if err != nil {
		test.Errorf("Failed to begin transaction: %v", err)
	}

	_, err = tx.Exec(
		"INSERT INTO accounts (platform, identifier, passphrase) VALUES (?, ?, ?)",
		"test",
		"user1",
		"pass1",
	)
	if err != nil {
		test.Errorf("Failed to insert in transaction: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		test.Errorf("Failed to commit transaction: %v", err)
	}

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM accounts WHERE platform = ? AND identifier = ?",
		"test",
		"user1",
	).Scan(&count)
	if err != nil {
		test.Errorf("Failed to verify committed data: %v", err)
	}

	if count != 1 {
		test.Errorf("Expected 1 account after commit, got %d", count)
	}
}

func TestDatabase_Rollback(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	initializeTablesForTest(test, db)

	tx, err := db.Begin()
	if err != nil {
		test.Errorf("Failed to begin transaction: %v", err)
	}

	_, err = tx.Exec(
		"INSERT INTO accounts (platform, identifier, passphrase) VALUES (?, ?, ?)",
		"test",
		"user2",
		"pass2",
	)
	if err != nil {
		test.Errorf("Failed to insert in transaction: %v", err)
	}

	err = tx.Rollback()
	if err != nil {
		test.Errorf("Failed to rollback transaction: %v", err)
	}

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM accounts WHERE platform = ? AND identifier = ?",
		"test",
		"user2",
	).Scan(&count)
	if err != nil {
		test.Errorf("Failed to verify rolled back data: %v", err)
	}

	if count != 0 {
		test.Errorf("Expected 0 accounts after rollback, got %d", count)
	}
}

func TestDatabase_PreparedStatements(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	initializeTablesForTest(test, db)

	stmt, err := db.Prepare(
		"INSERT INTO accounts (platform, identifier, passphrase) VALUES (?, ?, ?)",
	)
	if err != nil {
		test.Errorf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	platforms := []string{"gmail", "github", "twitter"}
	identifiers := []string{"user1", "user2", "user3"}
	passphrases := []string{"pass1", "pass2", "pass3"}

	for i := 0; i < len(platforms); i++ {
		_, err = stmt.Exec(platforms[i], identifiers[i], passphrases[i])
		if err != nil {
			test.Errorf("Failed to execute prepared statement: %v", err)
		}
	}

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM accounts WHERE platform IN (?, ?, ?)",
		platforms[0],
		platforms[1],
		platforms[2],
	).Scan(&count)
	if err != nil {
		test.Errorf("Failed to count inserted accounts: %v", err)
	}

	if count != 3 {
		test.Errorf("Expected 3 accounts, got %d", count)
	}
}

func TestDatabase_TimeoutHandling(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	done := make(chan bool, 1)
	go func() {
		err := db.Ping()
		if err != nil {
			test.Errorf("Ping failed: %v", err)
		}
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		test.Error("Database operation timed out")
	}
}

func TestDatabase_ErrorHandling(test *testing.T) {
	cleanupTestDB(test)

	db := GetDB()

	_, err := db.Exec("INVALID SQL STATEMENT")
	if err == nil {
		test.Error("Expected error for invalid SQL, got none")
	}

	_, err = db.Query("SELECT * FROM non_existent_table")
	if err == nil {
		test.Error("Expected error for non-existent table, got none")
	}

	initializeTablesForTest(test, db)
	_, err = db.Exec("INSERT INTO user (passphrase, recovery) VALUES (NULL, 'test')")
	if err == nil {
		test.Error("Expected error for NULL in NOT NULL column, got none")
	}
}

func initializeTablesForTest(test *testing.T, db *sql.DB) {
	queries := []string{
		QueryCreateUserTable,
		QueryCreateAccountsTable,
		QuerySeedUser,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			test.Fatalf("Failed to initialize tables: %v", err)
		}
	}
}

func cleanupTestDB(test *testing.T) {
	if instance != nil && instance.connection != nil {
		instance.Close()
	}

	dbPath := "database/passenger.db"
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		test.Errorf("Failed to remove test database: %v", err)
	}

	instance = nil
	once = sync.Once{}
}

func BenchmarkGetDB(benchmark *testing.B) {
	cleanupTestDB(&testing.T{})

	for benchmark.Loop() {
		GetDB()
	}
}

func BenchmarkDatabasePing(benchmark *testing.B) {
	cleanupTestDB(&testing.T{})
	db := GetDB()

	benchmark.ResetTimer()
	for benchmark.Loop() {
		if err := db.Ping(); err != nil {
			benchmark.Errorf("Ping failed: %v", err)
		}
	}
}

func BenchmarkDatabaseQuery(benchmark *testing.B) {
	cleanupTestDB(&testing.T{})
	db := GetDB()
	initializeTablesForTest(&testing.T{}, db)

	for benchmark.Loop() {
		rows, err := db.Query("SELECT COUNT(*) FROM user")
		if err != nil {
			benchmark.Errorf("Query failed: %v", err)
		}
		rows.Close()
	}
}

func BenchmarkDatabaseInsert(benchmark *testing.B) {
	cleanupTestDB(&testing.T{})
	db := GetDB()
	initializeTablesForTest(&testing.T{}, db)

	for index := 0; benchmark.Loop(); index++ {
		_, err := db.Exec(
			"INSERT INTO accounts (platform, identifier, passphrase) VALUES (?, ?, ?)",
			"benchmark",
			"user"+string(rune(index)),
			"pass"+string(rune(index)),
		)
		if err != nil {
			benchmark.Errorf("Insert failed: %v", err)
		}
	}
}
