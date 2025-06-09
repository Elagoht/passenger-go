package importer

import (
	"encoding/csv"
	"io"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/url"
)

type fieldTransformer func(string) string

type Platform struct {
	Fields          []string
	DelimiterType   string // "CRLF" or "LF" for line endings
	DelimiterQuotes string
	MatchFields     map[string]string
	TransformFields map[string]fieldTransformer
}

type PlatformType string

const (
	PlatformFirefox  PlatformType = "Firefox"
	PlatformChromium PlatformType = "Chromium"
)

var (
	PassengerFieldNames = []string{"platform", "identifier", "passphrase", "note", "favorite"}

	PlatformTypes = []PlatformType{PlatformFirefox, PlatformChromium}

	platforms = map[PlatformType]Platform{
		PlatformFirefox: {
			Fields:          []string{"url", "username", "password", "httpRealm", "formActionOrigin", "guid", "timeCreated", "timeLastUsed", "timePasswordChanged"},
			DelimiterType:   "CRLF",
			DelimiterQuotes: "\"",
			MatchFields: map[string]string{
				"platform":   "url",
				"identifier": "username",
				"passphrase": "password",
				"url":        "url",
			},
			TransformFields: map[string]fieldTransformer{
				"platform": url.ConvertURLToPlatformName,
			},
		},
		PlatformChromium: {
			Fields:          []string{"name", "url", "username", "password", "note"},
			DelimiterType:   "LF",
			DelimiterQuotes: "",
			MatchFields: map[string]string{
				"platform":   "name",
				"identifier": "username",
				"passphrase": "password",
				"url":        "url",
			},
			TransformFields: map[string]fieldTransformer{},
		},
	}
)

func determinePlatform(file io.Reader) (PlatformType, error) {
	csvReader := csv.NewReader(file)
	csvReader.TrimLeadingSpace = true

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

func GetPlatform(file io.Reader) Platform {
	platformType, err := determinePlatform(file)
	if err != nil {
		return Platform{}
	}

	return platforms[platformType]
}

func (p Platform) Parse(
	file io.Reader,
) ([]schemas.RequestAccountsCreate, error) {
	csvReader := csv.NewReader(file)
	csvReader.Comma = ',' // Always use comma as delimiter
	csvReader.LazyQuotes = p.DelimiterQuotes == ""
	csvReader.TrimLeadingSpace = true

	// Skip header row
	_, err := csvReader.Read()
	if err != nil {
		return nil, schemas.NewAPIError(
			schemas.ErrInvalidPlatform,
			"Failed to read CSV header",
			err,
		)
	}

	results := []schemas.RequestAccountsCreate{}

	// Find field indices
	platformIndex := findFieldIndex(p.Fields, p.MatchFields["platform"])
	usernameIndex := findFieldIndex(p.Fields, p.MatchFields["identifier"])
	passwordIndex := findFieldIndex(p.Fields, p.MatchFields["passphrase"])
	urlIndex := findFieldIndex(p.Fields, p.MatchFields["url"])

	if platformIndex == -1 || usernameIndex == -1 || passwordIndex == -1 || urlIndex == -1 {
		return nil, schemas.NewAPIError(
			schemas.ErrInvalidPlatform,
			"Required fields not found in CSV",
			nil,
		)
	}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, schemas.NewAPIError(
				schemas.ErrUnprocessableEntity,
				"Failed to read CSV record",
				err,
			)
		}

		if len(record) < len(p.Fields) {
			continue // Skip malformed rows
		}

		account := schemas.RequestAccountsCreate{
			Platform:   calculateFields(p.TransformFields["platform"], record[platformIndex]),
			Identifier: calculateFields(p.TransformFields["identifier"], record[usernameIndex]),
			Passphrase: calculateFields(p.TransformFields["passphrase"], record[passwordIndex]),
			Url:        calculateFields(p.TransformFields["url"], record[urlIndex]),
		}

		results = append(results, account)
	}

	return results, nil
}

func calculateFields(
	fieldTransformer any,
	field string,
) string {
	function, ok := fieldTransformer.(func(string) string)

	if !ok {
		return field
	}

	return function(field)
}

func findFieldIndex(fields []string, field string) int {
	for i, f := range fields {
		if f == field {
			return i
		}
	}

	return -1
}
