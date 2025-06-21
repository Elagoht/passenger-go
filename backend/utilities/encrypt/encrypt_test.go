package encrypt

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func init() {
	if os.Getenv("AES_GCM_SECRET") == "" {
		os.Setenv("AES_GCM_SECRET", "12345678901234567890123456789012")
	}
	if os.Getenv("SALT") == "" {
		os.Setenv("SALT", "test-salt-for-argon2")
	}
}

func TestMain(main *testing.M) {
	os.Setenv("AES_GCM_SECRET", "12345678901234567890123456789012")
	os.Setenv("SALT", "test-salt-for-argon2")
	InitAESGCMSecret()
	code := main.Run()
	os.Unsetenv("AES_GCM_SECRET")
	os.Unsetenv("SALT")
	os.Exit(code)
}

func TestHashPassword(test *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "testpassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false,
		},
		{
			name:     "special characters",
			password: "!@#$%^&*()_+-=[]{}|;:,.<>?",
			wantErr:  false,
		},
		{
			name:     "unicode characters",
			password: "测试密码123",
			wantErr:  false,
		},
	}

	for _, currentTest := range tests {
		test.Run(currentTest.name, func(test *testing.T) {
			hash, err := HashPassword(currentTest.password)

			if currentTest.wantErr {
				if err == nil {
					test.Errorf("HashPassword() expected error but got none")
				}
				return
			}

			if err != nil {
				test.Errorf("HashPassword() unexpected error: %v", err)
				return
			}

			if hash == "" {
				test.Errorf("HashPassword() returned empty hash")
			}

			if _, err := base64.StdEncoding.DecodeString(hash); err != nil {
				test.Errorf("HashPassword() returned invalid base64: %v", err)
			}
		})
	}
}

func TestVerifyPassword(test *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		test.Fatalf("Failed to hash password: %v", err)
	}

	valid, err := VerifyPassword(password, hash)
	if err != nil {
		test.Errorf("VerifyPassword() unexpected error: %v", err)
	}
	if !valid {
		test.Errorf("VerifyPassword() should return true for correct password")
	}

	valid, err = VerifyPassword("wrongpassword", hash)
	if err != nil {
		test.Errorf("VerifyPassword() unexpected error: %v", err)
	}
	if valid {
		test.Errorf("VerifyPassword() should return false for incorrect password")
	}

	valid, err = VerifyPassword("", hash)
	if err != nil {
		test.Errorf("VerifyPassword() unexpected error: %v", err)
	}
	if valid {
		test.Errorf("VerifyPassword() should return false for empty password")
	}
}

func TestEncryptDecrypt(test *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "simple text",
			data: "Hello, World!",
		},
		{
			name: "empty string",
			data: "",
		},
		{
			name: "special characters",
			data: "!@#$%^&*()_+-=[]{}|;:,.<>?",
		},
		{
			name: "unicode characters",
			data: "测试文本",
		},
		{
			name: "long text",
			data: strings.Repeat(
				"This is a long text for testing encryption. ",
				100,
			),
		},
		{
			name: "json data",
			data: `{"username": "test", "email": "test@example.com", "password": "secret123"}`,
		},
	}

	for _, currentTest := range tests {
		test.Run(currentTest.name, func(test *testing.T) {
			encrypted, err := Encrypt(currentTest.data)
			if err != nil {
				test.Errorf("Encrypt() error: %v", err)
				return
			}

			if encrypted == "" {
				test.Errorf("Encrypt() returned empty string")
				return
			}

			if _, err := base64.StdEncoding.DecodeString(encrypted); err != nil {
				test.Errorf("Encrypt() returned invalid base64: %v", err)
				return
			}

			decrypted, err := Decrypt(encrypted)
			if err != nil {
				test.Errorf("Decrypt() error: %v", err)
				return
			}

			if decrypted != currentTest.data {
				test.Errorf("Decrypt() = %v, want %v", decrypted, currentTest.data)
			}
		})
	}
}

func TestEncryptDeterministicDecryptDeterministic(test *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "simple text",
			data: "Hello, World!",
		},
		{
			name: "empty string",
			data: "",
		},
		{
			name: "special characters",
			data: "!@#$%^&*()_+-=[]{}|;:,.<>?",
		},
		{
			name: "unicode characters",
			data: "测试文本",
		},
	}

	for _, currentTest := range tests {
		test.Run(currentTest.name, func(test *testing.T) {
			encrypted1, err := EncryptDeterministic(currentTest.data)
			if err != nil {
				test.Errorf("EncryptDeterministic() error: %v", err)
				return
			}

			encrypted2, err := EncryptDeterministic(currentTest.data)
			if err != nil {
				test.Errorf("EncryptDeterministic() error on second call: %v", err)
				return
			}

			if encrypted1 != encrypted2 {
				test.Errorf(
					"Deterministic encryption should produce identical results, got %v and %v",
					encrypted1,
					encrypted2,
				)
			}

			decrypted, err := DecryptDeterministic(encrypted1)
			if err != nil {
				test.Errorf("DecryptDeterministic() error: %v", err)
				return
			}

			if decrypted != currentTest.data {
				test.Errorf(
					"DecryptDeterministic() = %v, want %v",
					decrypted,
					currentTest.data,
				)
			}
		})
	}
}

func TestGenerateRecoveryKey(test *testing.T) {
	tests := []struct {
		name       string
		passphrase string
		wantErr    bool
	}{
		{
			name:       "valid passphrase",
			passphrase: "my-secure-passphrase",
			wantErr:    false,
		},
		{
			name:       "empty passphrase",
			passphrase: "",
			wantErr:    false,
		},
		{
			name:       "special characters",
			passphrase: "!@#$%^&*()_+-=[]{}|;:,.<>?",
			wantErr:    false,
		},
		{
			name:       "unicode characters",
			passphrase: "测试密码",
			wantErr:    false,
		},
	}

	for _, currentTest := range tests {
		test.Run(currentTest.name, func(thisTest *testing.T) {
			key1, err := GenerateRecoveryKey(currentTest.passphrase)

			if currentTest.wantErr {
				if err == nil {
					thisTest.Errorf("GenerateRecoveryKey() expected error but got none")
				}
				return
			}

			if err != nil {
				thisTest.Errorf("GenerateRecoveryKey() unexpected error: %v", err)
				return
			}

			if key1 == "" {
				thisTest.Errorf("GenerateRecoveryKey() returned empty key")
				return
			}

			if _, err := base64.URLEncoding.DecodeString(key1); err != nil {
				thisTest.Errorf(
					"GenerateRecoveryKey() returned invalid base64 URL: %v",
					err,
				)
				return
			}

			key2, err := GenerateRecoveryKey(currentTest.passphrase)
			if err != nil {
				thisTest.Errorf("GenerateRecoveryKey() error on second call: %v", err)
				return
			}

			if key1 == key2 {
				thisTest.Errorf("GenerateRecoveryKey() should produce different keys for same passphrase due to random salt")
			}
		})
	}
}

func TestAESGCMEncryptDeterministicConsistency(test *testing.T) {
	data := "test data for deterministic encryption"

	encrypted1, err := aesGCMEncryptDeterministic([]byte(data))
	if err != nil {
		test.Fatalf("aesGCMEncryptDeterministic() error: %v", err)
	}

	encrypted2, err := aesGCMEncryptDeterministic([]byte(data))
	if err != nil {
		test.Fatalf("aesGCMEncryptDeterministic() error on second call: %v", err)
	}

	encrypted3, err := aesGCMEncryptDeterministic([]byte(data))
	if err != nil {
		test.Fatalf("aesGCMEncryptDeterministic() error on third call: %v", err)
	}

	if encrypted1 != encrypted2 || encrypted2 != encrypted3 {
		test.Errorf("Deterministic encryption should produce identical results")
	}
}

func TestAESGCMDecryptInvalidData(test *testing.T) {
	_, err := aesGCMDecrypt("invalid-base64!")
	if err == nil {
		test.Errorf("aesGCMDecrypt() should return error for invalid base64")
	}

	_, err = aesGCMDecrypt("")
	if err == nil {
		test.Errorf("aesGCMDecrypt() should return error for empty string")
	}

	shortData := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = aesGCMDecrypt(shortData)
	if err == nil {
		test.Errorf("aesGCMDecrypt() should return error for too short data")
	}
}

func TestAESGCMDecryptDeterministicInvalidData(test *testing.T) {
	_, err := aesGCMDecryptDeterministic("invalid-base64!")
	if err == nil {
		test.Errorf("aesGCMDecryptDeterministic() should return error for invalid base64")
	}

	_, err = aesGCMDecryptDeterministic("")
	if err == nil {
		test.Errorf("aesGCMDecryptDeterministic() should return error for empty string")
	}

	shortData := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = aesGCMDecryptDeterministic(shortData)
	if err == nil {
		test.Errorf("aesGCMDecryptDeterministic() should return error for too short data")
	}
}

func TestEncryptDecryptRoundTrip(test *testing.T) {
	originalData := "This is a test string with special chars: !@#$%^&*() and unicode: 测试"

	encrypted, err := Encrypt(originalData)
	if err != nil {
		test.Fatalf("Encrypt() error: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		test.Fatalf("Decrypt() error: %v", err)
	}

	if decrypted != originalData {
		test.Errorf("Round trip failed: got %v, want %v", decrypted, originalData)
	}

	encryptedDet, err := EncryptDeterministic(originalData)
	if err != nil {
		test.Fatalf("EncryptDeterministic() error: %v", err)
	}

	decryptedDet, err := DecryptDeterministic(encryptedDet)
	if err != nil {
		test.Fatalf("DecryptDeterministic() error: %v", err)
	}

	if decryptedDet != originalData {
		test.Errorf("Deterministic round trip failed: got %v, want %v", decryptedDet, originalData)
	}
}

func TestEncryptionSecurity(test *testing.T) {
	data := "sensitive data"

	encrypted1, err := Encrypt(data)
	if err != nil {
		test.Fatalf("Encrypt() error: %v", err)
	}

	encrypted2, err := Encrypt(data)
	if err != nil {
		test.Fatalf("Encrypt() error on second call: %v", err)
	}

	if encrypted1 == encrypted2 {
		test.Errorf("Regular encryption should produce different results due to random nonce")
	}

	detEncrypted1, err := EncryptDeterministic(data)
	if err != nil {
		test.Fatalf("EncryptDeterministic() error: %v", err)
	}

	detEncrypted2, err := EncryptDeterministic(data)
	if err != nil {
		test.Fatalf("EncryptDeterministic() error on second call: %v", err)
	}

	if detEncrypted1 != detEncrypted2 {
		test.Errorf("Deterministic encryption should produce identical results")
	}
}

func TestDecryptWithTamperedData(test *testing.T) {
	originalData := "test data"
	encrypted, err := Encrypt(originalData)
	if err != nil {
		test.Fatalf("Encrypt() error: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		test.Fatalf("Failed to decode encrypted data: %v", err)
	}

	if len(decoded) > 20 {
		decoded[20] ^= 1
	}

	tamperedEncrypted := base64.StdEncoding.EncodeToString(decoded)

	_, err = Decrypt(tamperedEncrypted)
	if err == nil {
		test.Errorf("Decrypt() should return error for tampered data")
	}
}

func TestDecryptDeterministicWithTamperedData(test *testing.T) {
	originalData := "test data"
	encrypted, err := EncryptDeterministic(originalData)
	if err != nil {
		test.Fatalf("EncryptDeterministic() error: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		test.Fatalf("Failed to decode encrypted data: %v", err)
	}

	if len(decoded) > 20 {
		decoded[20] ^= 1
	}

	tamperedEncrypted := base64.StdEncoding.EncodeToString(decoded)

	_, err = DecryptDeterministic(tamperedEncrypted)
	if err == nil {
		test.Errorf("DecryptDeterministic() should return error for tampered data")
	}
}

func TestLargeDataEncryption(test *testing.T) {
	largeData := strings.Repeat(
		"This is a very long string for testing encryption with large data. ",
		1000,
	)

	encrypted, err := Encrypt(largeData)
	if err != nil {
		test.Fatalf("Encrypt() error with large data: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		test.Fatalf("Decrypt() error with large data: %v", err)
	}

	if decrypted != largeData {
		test.Errorf("Large data round trip failed")
	}
}

func TestBinaryDataEncryption(test *testing.T) {
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC}
	dataString := string(binaryData)

	encrypted, err := Encrypt(dataString)
	if err != nil {
		test.Fatalf("Encrypt() error with binary data: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		test.Fatalf("Decrypt() error with binary data: %v", err)
	}

	if decrypted != dataString {
		test.Errorf("Binary data round trip failed")
	}
}

func TestConcurrentEncryption(test *testing.T) {
	data := "test data for concurrent encryption"
	numGoroutines := 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			encrypted, err := Encrypt(data)
			if err != nil {
				errors <- err
				return
			}
			results <- encrypted
		}()
	}

	encryptedResults := make([]string, 0, numGoroutines)
	for range numGoroutines {
		select {
		case result := <-results:
			encryptedResults = append(encryptedResults, result)
		case err := <-errors:
			test.Errorf("Concurrent encryption error: %v", err)
		}
	}

	seen := make(map[string]bool)
	for _, result := range encryptedResults {
		if seen[result] {
			test.Errorf("Concurrent encryption produced duplicate results")
		}
		seen[result] = true
	}
}

func TestConcurrentDeterministicEncryption(test *testing.T) {
	data := "test data for concurrent deterministic encryption"
	numGoroutines := 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			encrypted, err := EncryptDeterministic(data)
			if err != nil {
				errors <- err
				return
			}
			results <- encrypted
		}()
	}

	encryptedResults := make([]string, 0, numGoroutines)
	for range numGoroutines {
		select {
		case result := <-results:
			encryptedResults = append(encryptedResults, result)
		case err := <-errors:
			test.Errorf("Concurrent deterministic encryption error: %v", err)
		}
	}

	firstResult := encryptedResults[0]
	for index, result := range encryptedResults {
		if result != firstResult {
			test.Errorf(
				"Concurrent deterministic encryption produced different results at index %d",
				index,
			)
		}
	}
}

func BenchmarkHashPassword(benchmark *testing.B) {
	password := "testpassword123"
	for benchmark.Loop() {
		_, err := HashPassword(password)
		if err != nil {
			benchmark.Fatalf("HashPassword() error: %v", err)
		}
	}
}

func BenchmarkEncrypt(benchmark *testing.B) {
	data := "test data for encryption benchmark"
	for benchmark.Loop() {
		_, err := Encrypt(data)
		if err != nil {
			benchmark.Fatalf("Encrypt() error: %v", err)
		}
	}
}

func BenchmarkDecrypt(benchmark *testing.B) {
	data := "test data for decryption benchmark"
	encrypted, err := Encrypt(data)
	if err != nil {
		benchmark.Fatalf("Failed to encrypt data for benchmark: %v", err)
	}

	for benchmark.Loop() {
		_, err := Decrypt(encrypted)
		if err != nil {
			benchmark.Fatalf("Decrypt() error: %v", err)
		}
	}
}

func BenchmarkEncryptDeterministic(benchmark *testing.B) {
	data := "test data for deterministic encryption benchmark"
	for benchmark.Loop() {
		_, err := EncryptDeterministic(data)
		if err != nil {
			benchmark.Fatalf("EncryptDeterministic() error: %v", err)
		}
	}
}

func BenchmarkGenerateRecoveryKey(benchmark *testing.B) {
	passphrase := "test passphrase for recovery key benchmark"
	for benchmark.Loop() {
		_, err := GenerateRecoveryKey(passphrase)
		if err != nil {
			benchmark.Fatalf("GenerateRecoveryKey() error: %v", err)
		}
	}
}
