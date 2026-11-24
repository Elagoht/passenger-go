package transfer

import (
	"encoding/csv"
	"io"

	"passenger-go/internal/store"
)

var exportHeader = []string{"platform", "identifier", "passphrase", "url", "notes"}

// Export writes accounts as Passenger CSV, which Import reads back.
func Export(w io.Writer, accounts []store.Account) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(exportHeader); err != nil {
		return err
	}
	for _, a := range accounts {
		if err := cw.Write([]string{a.Platform, a.Identifier, a.Passphrase, a.URL, a.Notes}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
