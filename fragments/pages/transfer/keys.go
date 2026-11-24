// Package fragments holds the content of the import and export pages.
package fragments

import "github.com/Elagoht/collage/pkg/collage"

// FailedRow is an imported row that was not stored. It carries no passphrase.
type FailedRow struct {
	Platform   string
	Identifier string
	Reason     string
}

// ImportResult is what an import did, shown on the page that answers it.
type ImportResult struct {
	Format   string
	Imported int
	Failed   []FailedRow
}

// Imported carries the result of an import to the page.
var Imported = collage.NewKey[ImportResult]("import-result")
