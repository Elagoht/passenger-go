package utilities

const (
	QueryCreateUserTable string = /* A single user table */ `
	CREATE TABLE IF NOT EXISTS user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		passphrase TEXT NOT NULL UNIQUE,
		recovery TEXT NOT NULL UNIQUE,
		validated BOOLEAN DEFAULT FALSE
	)
	`
	QueryCreateAccountsTable string = `
	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		identifier TEXT NOT NULL,
		passphrase TEXT NOT NULL,
		icon INT NOT NULL, -- This is an index of the icon set
		notes TEXT DEFAULT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME DEFAULT NULL, -- using soft delete
		favorite BOOLEAN DEFAULT FALSE
	)
	`
	QuerySeedUser = `
	INSERT INTO user (passphrase, recovery)
	SELECT '', ''
	WHERE NOT EXISTS (SELECT 1 FROM user)
	`
)
