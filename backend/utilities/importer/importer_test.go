package importer

import (
	"passenger-go/backend/schemas"
	"strings"
	"testing"
)

func TestDeterminePlatform_Firefox(test *testing.T) {
	firefoxCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged
https://example.com,testuser,testpass,,,guid123,1234567890,1234567890,1234567890`

	reader := strings.NewReader(firefoxCSV)
	platformType, err := determinePlatform(reader)

	if err != nil {
		test.Errorf("Expected no error, got %v", err)
	}

	if platformType != PlatformFirefox {
		test.Errorf(
			"Expected platform type %s, got %s",
			PlatformFirefox,
			platformType,
		)
	}
}

func TestDeterminePlatform_Chromium(test *testing.T) {
	chromiumCSV := `name,url,username,password,note
Example Site,https://example.com,testuser,testpass,Test note`

	reader := strings.NewReader(chromiumCSV)
	platformType, err := determinePlatform(reader)

	if err != nil {
		test.Errorf("Expected no error, got %v", err)
	}

	if platformType != PlatformChromium {
		test.Errorf("Expected platform type %s, got %s", PlatformChromium, platformType)
	}
}

func TestDeterminePlatform_InvalidCSV(test *testing.T) {
	invalidCSV := `invalid,header,row`

	reader := strings.NewReader(invalidCSV)
	platformType, err := determinePlatform(reader)

	if err == nil {
		test.Error("Expected error for invalid CSV, got nil")
	}

	if platformType != "" {
		test.Errorf("Expected empty platform type, got %s", platformType)
	}

	apiError, ok := err.(*schemas.APIError)
	if !ok {
		test.Error("Expected APIError type")
	}

	if apiError.Code != string(schemas.ErrInvalidPlatform) {
		test.Errorf(
			"Expected error code %s, got %s",
			schemas.ErrInvalidPlatform,
			apiError.Code,
		)
	}
}

func TestDeterminePlatform_EmptyFile(test *testing.T) {
	emptyFile := ""

	reader := strings.NewReader(emptyFile)
	platformType, err := determinePlatform(reader)

	if err == nil {
		test.Error("Expected error for empty file, got nil")
	}

	if platformType != "" {
		test.Errorf("Expected empty platform type, got %s", platformType)
	}
}

func TestDeterminePlatform_PartialMatch(test *testing.T) {
	partialCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged,extra_field
https://example.com,testuser,testpass,,,guid123,1234567890,1234567890,1234567890,extra`

	reader := strings.NewReader(partialCSV)
	platformType, err := determinePlatform(reader)

	if err == nil {
		test.Error("Expected error for partial match, got nil")
	}

	if platformType != "" {
		test.Errorf("Expected empty platform type, got %s", platformType)
	}
}

func TestGetPlatform_ValidFirefox(test *testing.T) {
	firefoxCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged
https://example.com,testuser,testpass,,,guid123,1234567890,1234567890,1234567890`

	reader := strings.NewReader(firefoxCSV)
	platform := GetPlatform(reader)

	if platform.Fields == nil {
		test.Error("Expected platform fields, got nil")
	}

	expectedFields := []string{
		"url",
		"username",
		"password",
		"httpRealm",
		"formActionOrigin",
		"guid",
		"timeCreated",
		"timeLastUsed",
		"timePasswordChanged",
	}
	if len(platform.Fields) != len(expectedFields) {
		test.Errorf(
			"Expected %d fields, got %d",
			len(expectedFields),
			len(platform.Fields),
		)
	}

	for index, field := range expectedFields {
		if platform.Fields[index] != field {
			test.Errorf(
				"Expected field %s at index %d, got %s",
				field,
				index,
				platform.Fields[index],
			)
		}
	}

	if platform.DelimiterType != "CRLF" {
		test.Errorf(
			"Expected delimiter type CRLF, got %s",
			platform.DelimiterType,
		)
	}

	if platform.DelimiterQuotes != "\"" {
		test.Errorf(
			"Expected delimiter quotes \", got %s",
			platform.DelimiterQuotes,
		)
	}
}

func TestGetPlatform_ValidChromium(test *testing.T) {
	chromiumCSV := `name,url,username,password,note
Example Site,https://example.com,testuser,testpass,Test note`

	reader := strings.NewReader(chromiumCSV)
	platform := GetPlatform(reader)

	if platform.Fields == nil {
		test.Error("Expected platform fields, got nil")
	}

	expectedFields := []string{
		"name",
		"url",
		"username",
		"password",
		"note",
	}
	if len(platform.Fields) != len(expectedFields) {
		test.Errorf(
			"Expected %d fields, got %d",
			len(expectedFields),
			len(platform.Fields),
		)
	}

	for index, field := range expectedFields {
		if platform.Fields[index] != field {
			test.Errorf(
				"Expected field %s at index %d, got %s",
				field,
				index,
				platform.Fields[index],
			)
		}
	}

	if platform.DelimiterType != "LF" {
		test.Errorf("Expected delimiter type LF, got %s", platform.DelimiterType)
	}

	if platform.DelimiterQuotes != "" {
		test.Errorf("Expected empty delimiter quotes, got %s", platform.DelimiterQuotes)
	}
}

func TestGetPlatform_InvalidPlatform(test *testing.T) {
	invalidCSV := `invalid,header,row`

	reader := strings.NewReader(invalidCSV)
	platform := GetPlatform(reader)

	if platform.Fields != nil {
		test.Error("Expected nil fields for invalid platform, got non-nil")
	}
}

func TestPlatform_Parse_Firefox(test *testing.T) {
	firefoxCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged
https://github.com,testuser,testpass,,,guid123,1234567890,1234567890,1234567890
https://google.com,user2,pass2,,,guid456,1234567890,1234567890,1234567890`

	reader := strings.NewReader(firefoxCSV)
	platform := platforms[PlatformFirefox]
	accounts, err := platform.Parse(reader)

	if err != nil {
		test.Errorf("Expected no error, got %v", err)
	}

	if len(accounts) != 2 {
		test.Errorf("Expected 2 accounts, got %d", len(accounts))
	}

	firstAccount := accounts[0]
	if firstAccount.Platform != "https://github.com" {
		test.Errorf("Expected platform https://github.com, got %s", firstAccount.Platform)
	}
	if firstAccount.Identifier != "testuser" {
		test.Errorf("Expected identifier testuser, got %s", firstAccount.Identifier)
	}
	if firstAccount.Passphrase != "testpass" {
		test.Errorf("Expected passphrase testpass, got %s", firstAccount.Passphrase)
	}
	if firstAccount.Url != "https://github.com" {
		test.Errorf("Expected URL https://github.com, got %s", firstAccount.Url)
	}

	secondAccount := accounts[1]
	if secondAccount.Platform != "https://google.com" {
		test.Errorf(
			"Expected platform https://google.com, got %s",
			secondAccount.Platform,
		)
	}
	if secondAccount.Identifier != "user2" {
		test.Errorf("Expected identifier user2, got %s",
			secondAccount.Identifier,
		)
	}
	if secondAccount.Passphrase != "pass2" {
		test.Errorf("Expected passphrase pass2, got %s", secondAccount.Passphrase)
	}
	if secondAccount.Url != "https://google.com" {
		test.Errorf("Expected URL https://google.com, got %s", secondAccount.Url)
	}
}

func TestPlatform_Parse_Chromium(test *testing.T) {
	chromiumCSV := `name,url,username,password,note
GitHub,https://github.com,testuser,testpass,Test note
Google,https://google.com,user2,pass2,Another note`

	reader := strings.NewReader(chromiumCSV)
	platform := platforms[PlatformChromium]
	accounts, err := platform.Parse(reader)

	if err != nil {
		test.Errorf("Expected no error, got %v", err)
	}

	if len(accounts) != 2 {
		test.Errorf("Expected 2 accounts, got %d", len(accounts))
	}

	firstAccount := accounts[0]
	if firstAccount.Platform != "GitHub" {
		test.Errorf("Expected platform GitHub, got %s", firstAccount.Platform)
	}
	if firstAccount.Identifier != "testuser" {
		test.Errorf(
			"Expected identifier testuser, got %s",
			firstAccount.Identifier,
		)
	}
	if firstAccount.Passphrase != "testpass" {
		test.Errorf(
			"Expected passphrase testpass, got %s",
			firstAccount.Passphrase,
		)
	}
	if firstAccount.Url != "https://github.com" {
		test.Errorf("Expected URL https://github.com, got %s", firstAccount.Url)
	}
	if firstAccount.Notes != "Test note" {
		test.Errorf("Expected notes 'Test note', got %s", firstAccount.Notes)
	}

	secondAccount := accounts[1]
	if secondAccount.Platform != "Google" {
		test.Errorf("Expected platform Google, got %s", secondAccount.Platform)
	}
	if secondAccount.Identifier != "user2" {
		test.Errorf("Expected identifier user2, got %s", secondAccount.Identifier)
	}
	if secondAccount.Passphrase != "pass2" {
		test.Errorf("Expected passphrase pass2, got %s", secondAccount.Passphrase)
	}
	if secondAccount.Url != "https://google.com" {
		test.Errorf("Expected URL https://google.com, got %s", secondAccount.Url)
	}
	if secondAccount.Notes != "Another note" {
		test.Errorf("Expected notes 'Another note', got %s", secondAccount.Notes)
	}
}

func TestPlatform_Parse_EmptyFile(test *testing.T) {
	emptyCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged`

	reader := strings.NewReader(emptyCSV)
	platform := platforms[PlatformFirefox]
	accounts, err := platform.Parse(reader)

	if err != nil {
		test.Errorf("Expected no error for empty file, got %v", err)
	}

	if len(accounts) != 0 {
		test.Errorf("Expected 0 accounts, got %d", len(accounts))
	}
}

func TestPlatform_Parse_MalformedRow(test *testing.T) {
	malformedCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged
https://github.com,testuser,testpass,,,guid123,1234567890,1234567890,1234567890
https://google.com,user2
https://example.com,user3,pass3,,,guid789,1234567890,1234567890,1234567890`

	reader := strings.NewReader(malformedCSV)
	platform := platforms[PlatformFirefox]
	_, err := platform.Parse(reader)

	if err == nil {
		test.Error("Expected error for malformed CSV, got nil")
		return
	}

	apiError, ok := err.(*schemas.APIError)
	if !ok {
		test.Error("Expected APIError type")
		return
	}

	if apiError.Code != string(schemas.ErrUnprocessableEntity) {
		test.Errorf(
			"Expected error code %s, got %s",
			schemas.ErrUnprocessableEntity,
			apiError.Code,
		)
	}
}

func TestPlatform_Parse_MissingRequiredFields(test *testing.T) {
	invalidPlatform := Platform{
		Fields: []string{"url", "username", "password"},
		MatchFields: map[string]string{
			"platform":   "url",
			"identifier": "username",
		},
		TransformFields: map[string]fieldTransformer{},
	}

	validCSV := `url,username,password
https://github.com,testuser,testpass`

	reader := strings.NewReader(validCSV)
	accounts, err := invalidPlatform.Parse(reader)

	if err == nil {
		test.Error("Expected error for missing required fields, got nil")
	}

	if accounts != nil {
		test.Error("Expected nil accounts, got non-nil")
	}

	apiError, ok := err.(*schemas.APIError)
	if !ok {
		test.Error("Expected APIError type")
	}

	if apiError.Code != string(schemas.ErrInvalidPlatform) {
		test.Errorf(
			"Expected error code %s, got %s",
			schemas.ErrInvalidPlatform,
			apiError.Code,
		)
	}
}

func TestPlatform_Parse_InvalidCSV(test *testing.T) {
	invalidCSV := `invalid,header,row`

	reader := strings.NewReader(invalidCSV)
	platform := platforms[PlatformFirefox]
	accounts, err := platform.Parse(reader)

	if err != nil {
		test.Errorf("Expected no error for invalid CSV, got %v", err)
		return
	}

	if len(accounts) != 0 {
		test.Errorf("Expected 0 accounts, got %d", len(accounts))
	}
}

func TestCalculateFields_WithTransformer(test *testing.T) {
	transformer := func(s string) string {
		return strings.ToUpper(s)
	}

	result := calculateFields(transformer, "test")
	expected := "TEST"

	if result != expected {
		test.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCalculateFields_WithoutTransformer(test *testing.T) {
	result := calculateFields(nil, "test")
	expected := "test"

	if result != expected {
		test.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCalculateFields_WithInvalidTransformer(test *testing.T) {
	invalidTransformer := "not a function"
	result := calculateFields(invalidTransformer, "test")
	expected := "test"

	if result != expected {
		test.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCalculateFields_WithEmptyString(test *testing.T) {
	transformer := func(s string) string {
		return strings.ToUpper(s)
	}

	result := calculateFields(transformer, "")
	expected := ""

	if result != expected {
		test.Errorf("Expected empty string, got %s", result)
	}
}

func TestFindFieldIndex_ExistingField(test *testing.T) {
	fields := []string{"url", "username", "password", "note"}

	index := findFieldIndex(fields, "username")
	expected := 1

	if index != expected {
		test.Errorf("Expected index %d, got %d", expected, index)
	}
}

func TestFindFieldIndex_FirstField(test *testing.T) {
	fields := []string{"url", "username", "password", "note"}

	index := findFieldIndex(fields, "url")
	expected := 0

	if index != expected {
		test.Errorf("Expected index %d, got %d", expected, index)
	}
}

func TestFindFieldIndex_LastField(test *testing.T) {
	fields := []string{"url", "username", "password", "note"}

	index := findFieldIndex(fields, "note")
	expected := 3

	if index != expected {
		test.Errorf("Expected index %d, got %d", expected, index)
	}
}

func TestFindFieldIndex_NonExistentField(test *testing.T) {
	fields := []string{"url", "username", "password", "note"}

	index := findFieldIndex(fields, "nonexistent")
	expected := -1

	if index != expected {
		test.Errorf("Expected index %d, got %d", expected, index)
	}
}

func TestFindFieldIndex_EmptySlice(test *testing.T) {
	fields := []string{}

	index := findFieldIndex(fields, "url")
	expected := -1

	if index != expected {
		test.Errorf("Expected index %d, got %d", expected, index)
	}
}

func TestFindFieldIndex_CaseSensitive(test *testing.T) {
	fields := []string{"URL", "Username", "Password", "Note"}

	index := findFieldIndex(fields, "url")
	expected := -1

	if index != expected {
		test.Errorf("Expected index %d, got %d", expected, index)
	}
}

func TestPlatformConstants(test *testing.T) {
	if PlatformFirefox != "Firefox" {
		test.Errorf(
			"Expected PlatformFirefox to be 'Firefox', got %s",
			PlatformFirefox,
		)
	}

	if PlatformChromium != "Chromium" {
		test.Errorf(
			"Expected PlatformChromium to be 'Chromium', got %s",
			PlatformChromium,
		)
	}

	if len(PlatformTypes) != 2 {
		test.Errorf("Expected 2 platform types, got %d", len(PlatformTypes))
	}

	foundFirefox := false
	foundChromium := false
	for _, pt := range PlatformTypes {
		if pt == PlatformFirefox {
			foundFirefox = true
		}
		if pt == PlatformChromium {
			foundChromium = true
		}
	}

	if !foundFirefox {
		test.Error("PlatformFirefox not found in PlatformTypes")
	}

	if !foundChromium {
		test.Error("PlatformChromium not found in PlatformTypes")
	}
}

func TestPassengerFieldNames(test *testing.T) {
	expectedFields := []string{
		"platform",
		"identifier",
		"passphrase",
		"note",
		"favorite",
	}

	if len(PassengerFieldNames) != len(expectedFields) {
		test.Errorf(
			"Expected %d passenger field names, got %d",
			len(expectedFields),
			len(PassengerFieldNames),
		)
	}

	for index, field := range expectedFields {
		if PassengerFieldNames[index] != field {
			test.Errorf(
				"Expected field %s at index %d, got %s",
				field,
				index,
				PassengerFieldNames[index],
			)
		}
	}
}

func TestPlatform_Parse_WithQuotes(test *testing.T) {
	quotedCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged
"https://github.com","test,user","test,pass",,,guid123,1234567890,1234567890,1234567890`

	reader := strings.NewReader(quotedCSV)
	platform := platforms[PlatformFirefox]
	accounts, err := platform.Parse(reader)

	if err != nil {
		test.Errorf("Expected no error, got %v", err)
	}

	if len(accounts) != 1 {
		test.Errorf("Expected 1 account, got %d", len(accounts))
	}

	account := accounts[0]
	if account.Identifier != "test,user" {
		test.Errorf("Expected identifier 'test,user', got %s", account.Identifier)
	}
	if account.Passphrase != "test,pass" {
		test.Errorf("Expected passphrase 'test,pass', got %s", account.Passphrase)
	}
}

func TestPlatform_Parse_WithEmptyFields(test *testing.T) {
	emptyFieldsCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged
https://github.com,,testpass,,,guid123,1234567890,1234567890,1234567890
https://google.com,user2,,,guid456,1234567890,1234567890,1234567890`

	reader := strings.NewReader(emptyFieldsCSV)
	platform := platforms[PlatformFirefox]
	_, err := platform.Parse(reader)

	if err == nil {
		test.Error("Expected error for malformed row with empty fields, got nil")
		return
	}

	apiError, ok := err.(*schemas.APIError)
	if !ok {
		test.Error("Expected APIError type")
		return
	}

	if apiError.Code != string(schemas.ErrUnprocessableEntity) {
		test.Errorf(
			"Expected error code %s, got %s",
			schemas.ErrUnprocessableEntity,
			apiError.Code,
		)
	}
}

func BenchmarkDeterminePlatform_Firefox(benchmark *testing.B) {
	firefoxCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged
https://example.com,testuser,testpass,,,guid123,1234567890,1234567890,1234567890`

	for benchmark.Loop() {
		reader := strings.NewReader(firefoxCSV)
		determinePlatform(reader)
	}
}

func BenchmarkDeterminePlatform_Chromium(benchmark *testing.B) {
	chromiumCSV := `name,url,username,password,note
Example Site,https://example.com,testuser,testpass,Test note`

	for benchmark.Loop() {
		reader := strings.NewReader(chromiumCSV)
		_, _ = determinePlatform(reader)
	}
}

func BenchmarkPlatform_Parse_Firefox(benchmark *testing.B) {
	firefoxCSV := `url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged
https://github.com,testuser,testpass,,,guid123,1234567890,1234567890,1234567890
https://google.com,user2,pass2,,,guid456,1234567890,1234567890,1234567890`

	platform := platforms[PlatformFirefox]

	for benchmark.Loop() {
		reader := strings.NewReader(firefoxCSV)
		_, _ = platform.Parse(reader)
	}
}

func BenchmarkFindFieldIndex(benchmark *testing.B) {
	fields := []string{
		"url",
		"username",
		"password",
		"note",
		"favorite",
		"created",
		"updated",
	}

	for benchmark.Loop() {
		_ = findFieldIndex(fields, "password")
	}
}
