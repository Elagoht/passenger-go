package transfer

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"

	"passenger-go/internal/vault"
)

// ErrUnknownFormat means the file's header is none of the supported ones.
var ErrUnknownFormat = errors.New("transfer: unrecognised CSV format")

// Format names, as Import reports them.
const (
	FormatFirefox   = "Firefox"
	FormatChromium  = "Chromium"
	FormatPassenger = "Passenger"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// format maps a CSV header to records.
type format struct {
	name   string
	header []string
	record func(row []string) vault.Record
}

var formats = []format{
	{
		FormatFirefox,
		[]string{"url", "username", "password", "httpRealm", "formActionOrigin", "guid", "timeCreated", "timeLastUsed", "timePasswordChanged"},
		func(r []string) vault.Record {
			return vault.Record{Platform: PlatformFromURL(r[0]), Identifier: r[1], Passphrase: r[2], URL: r[0]}
		},
	},
	{
		FormatChromium,
		[]string{"name", "url", "username", "password", "note"},
		func(r []string) vault.Record {
			return vault.Record{Platform: r[0], URL: r[1], Identifier: r[2], Passphrase: r[3], Notes: r[4]}
		},
	},
	{
		FormatPassenger,
		exportHeader,
		func(r []string) vault.Record {
			return vault.Record{Platform: r[0], Identifier: r[1], Passphrase: r[2], URL: r[3], Notes: r[4]}
		},
	},
}

// Import reads a Firefox, Chromium or Passenger CSV export, detected by its
// exact header (a leading UTF-8 BOM is ignored), and returns the format's name
// and a record per row. Rows with fewer fields than the header are skipped.
// Records are returned as read: it is the caller's job to refuse incomplete
// ones.
func Import(r io.Reader) (string, []vault.Record, error) {
	br := bufio.NewReader(r)
	if head, err := br.Peek(len(utf8BOM)); err == nil && bytes.Equal(head, utf8BOM) {
		_, _ = br.Discard(len(utf8BOM))
	}
	cr := csv.NewReader(br)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return "", nil, ErrUnknownFormat
	}
	if err != nil {
		return "", nil, fmt.Errorf("transfer: read header: %w", err)
	}
	i := slices.IndexFunc(formats, func(f format) bool { return slices.Equal(f.header, header) })
	if i < 0 {
		return "", nil, ErrUnknownFormat
	}
	f := formats[i]

	records := []vault.Record{}
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return f.name, records, nil
		}
		if err != nil {
			return "", nil, fmt.Errorf("transfer: read row: %w", err)
		}
		if len(row) < len(f.header) {
			continue
		}
		records = append(records, f.record(row))
	}
}
