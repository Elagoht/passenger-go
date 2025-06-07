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
		url TEXT DEFAULT NULL,
		passphrase TEXT NOT NULL,
		notes TEXT DEFAULT NULL,
		favorite BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME DEFAULT NULL, -- using soft delete
		access_count INTEGER DEFAULT 0,
		strength INTEGER DEFAULT 0,
		last_accessed DATETIME DEFAULT NULL,
		UNIQUE(platform, identifier)
	)
	`
	QuerySeedUser = `
	INSERT INTO user (passphrase, recovery)
	SELECT '', ''
	WHERE NOT EXISTS (SELECT 1 FROM user)
	`
)
