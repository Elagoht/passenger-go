package repositories

const (
	QueryAccountCreate = `
	INSERT INTO accounts (platform, identifier, passphrase, url, notes, strength)
	VALUES (?, ?, ?, ?, ?, ?)
	`
	QueryAccounts = `
	SELECT id, platform, identifier, url, notes, strength
	FROM accounts
	`
	QueryAccountDetails = `
	SELECT id, platform, identifier, url, passphrase, notes, strength
	FROM accounts
	WHERE id = ?
	`
	QueryAccountPassphrase = `
	SELECT passphrase
	FROM accounts
	WHERE id = ?
	`
	QueryAccountUpdate = `
	UPDATE accounts
	SET platform = ?, identifier = ?, passphrase = ?, url = ?, notes = ?, strength = ?
	WHERE id = ?
	`
	QueryAccountDelete = `
	DELETE FROM accounts
	WHERE id = ?
	`
	QueryAccountImport = `
	BEGIN TRANSACTION;
	INSERT INTO accounts (platform, identifier, passphrase, url, notes, strength)
	VALUES (?, ?, ?, ?, ?, ?);
	COMMIT;
	`
	QueryAccountsExport = `
	SELECT platform, identifier, passphrase, url, notes
	FROM accounts
	`
	QueryUniqueIdentifiers = `
	SELECT DISTINCT identifier
	FROM accounts
	WHERE identifier IS NOT NULL AND identifier != ''
	ORDER BY identifier
	`
)
