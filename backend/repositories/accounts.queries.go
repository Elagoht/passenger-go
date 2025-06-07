package repositories

const (
	QueryAccountCreate = `
	INSERT INTO accounts (platform, identifier, passphrase, url, notes, favorite)
	VALUES (?, ?, ?, ?, ?, ?)
	`
	QueryAccountCards = `
	SELECT id, platform, identifier, url, favorite
	FROM accounts
	WHERE deleted_at IS NULL
	ORDER BY favorite DESC, created_at DESC
	LIMIT ? OFFSET ?
	`
	QueryAccountDetails = `
	SELECT id, platform, identifier, url, notes, favorite, created_at, updated_at, access_count, strength, last_accessed
	FROM accounts
	WHERE id = ? AND deleted_at IS NULL
	`
	QueryAccountUpdate = `
	UPDATE accounts
	SET platform = ?, identifier = ?, passphrase = ?, url = ?, notes = ?, favorite = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`
	QueryAccountDelete = `
	UPDATE accounts
	SET deleted_at = CURRENT_TIMESTAMP
	WHERE id = ? AND deleted_at IS NULL
	`
	QueryAccountAccessed = `
	UPDATE accounts
	SET access_count = access_count + 1, last_accessed = CURRENT_TIMESTAMP
	WHERE id = ? AND deleted_at IS NULL
	`
	QueryAccountPassphrase = `
	SELECT passphrase
	FROM accounts
	WHERE id = ? AND deleted_at IS NULL
	`
	QueryAccountImport = `
	BEGIN TRANSACTION;
	INSERT INTO accounts (platform, identifier, passphrase, url, notes)
	VALUES (?, ?, ?, ?, ?);
	COMMIT;
	`
)
