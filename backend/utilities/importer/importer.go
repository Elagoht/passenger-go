package importer

import (
	"encoding/csv"
	"io"
	"passenger-go/backend/schemas"
)

type Platform struct {
	Fields          []string
	DelimiterType   string
	DelimiterQuotes string
}

type PlatformType string

const (
	PlatformFirefox  PlatformType = "Firefox"
	PlatformChromium PlatformType = "Chromium"
)

var (
	PlatformTypes = []PlatformType{PlatformFirefox, PlatformChromium}

	platforms = map[PlatformType]Platform{
		PlatformFirefox: {
			Fields:          []string{"url", "username", "password", "httpRealm", "formActionOrigin", "guid", "timeCreated", "timeLastUsed", "timePasswordChanged"},
			DelimiterType:   "CRLF",
			DelimiterQuotes: "\"",
		},
		PlatformChromium: {
			Fields:          []string{"name", "url", "username", "password", "note"},
			DelimiterType:   "LF",
			DelimiterQuotes: "",
		},
	}
)

func DeterminePlatform(file io.Reader) (PlatformType, error) {
	csvReader := csv.NewReader(file)

	// Read the header row
	fields, err := csvReader.Read()
	if err != nil {
		return "", schemas.NewAPIError(
			schemas.ErrInvalidPlatform,
			"Failed to read CSV header",
			err,
		)
	}

	// Check each platform for a match
	for _, platformType := range PlatformTypes {
		platform := platforms[platformType]

		if len(fields) != len(platform.Fields) {
			continue
		}

		fieldsMatch := true
		for i, field := range platform.Fields {
			if fields[i] != field {
				fieldsMatch = false
				break
			}
		}

		if fieldsMatch {
			return platformType, nil
		}
	}

	// If no match is found, return an error
	return "", schemas.NewAPIError(
		schemas.ErrInvalidPlatform,
		"File does not match any of the supported platforms",
		nil,
	)
}
