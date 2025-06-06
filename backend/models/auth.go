package models

type User struct {
	Id         int
	Passphrase string
	Recovery   string
	Validated  bool
}
