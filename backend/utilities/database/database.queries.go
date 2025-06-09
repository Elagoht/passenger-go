package database

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
		strength INTEGER DEFAULT 0,
		UNIQUE(platform, identifier)
	)
	`
	QuerySeedUser = `
	INSERT INTO user (passphrase, recovery)
	SELECT '', ''
	WHERE NOT EXISTS (SELECT 1 FROM user)
	`
)
