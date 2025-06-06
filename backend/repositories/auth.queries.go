package repositories

const (
	QueryGetUserCount     = `SELECT COUNT(*) FROM user WHERE validated = TRUE`
	QueryGetUser          = `SELECT id, passphrase, validated, recovery FROM user LIMIT 1`
	QueryCreateUser       = `UPDATE user SET passphrase = ?, recovery = ? WHERE id = 1`
	QueryValidateUser     = `UPDATE user SET validated = TRUE`
	QueryUpdatePassphrase = `UPDATE user SET passphrase = ?`
)
