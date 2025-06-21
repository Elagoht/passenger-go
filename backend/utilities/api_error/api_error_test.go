package api_error

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"passenger-go/backend/schemas"
	"strings"
	"testing"
)

func TestHandleAPIError_WithAPIError_DevelopmentMode(test *testing.T) {
	os.Setenv("MODE", "development")
	defer os.Unsetenv("MODE")

	testError := schemas.NewAPIError(
		schemas.ErrInvalidCredentials,
		"Invalid username or password",
		errors.New("test stack error"),
	)

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, testError)

	if status := responseRecorder.Code; status != http.StatusUnauthorized {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusUnauthorized,
		)
	}

	expected := `{"code":"INVALID_CREDENTIALS","message":"Invalid username or password"}`
	if strings.TrimSpace(responseRecorder.Body.String()) != expected {
		test.Errorf(
			"HandleAPIError returned unexpected body: got %v want %v",
			responseRecorder.Body.String(),
			expected,
		)
	}
}

func TestHandleAPIError_WithAPIError_ProductionMode(test *testing.T) {
	os.Setenv("MODE", "production")
	defer os.Unsetenv("MODE")

	testError := schemas.NewAPIError(
		schemas.ErrInvalidCredentials,
		"Invalid username or password",
		errors.New("test stack error"),
	)

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, testError)

	if status := responseRecorder.Code; status != http.StatusUnauthorized {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusUnauthorized,
		)
	}

	expected := `{"code":"INVALID_CREDENTIALS","message":"Invalid username or password"}`
	if strings.TrimSpace(responseRecorder.Body.String()) != expected {
		test.Errorf(
			"HandleAPIError returned unexpected body: got %v want %v",
			responseRecorder.Body.String(),
			expected,
		)
	}
}

func TestHandleAPIError_WithRegularError_DevelopmentMode(test *testing.T) {
	os.Setenv("MODE", "development")
	defer os.Unsetenv("MODE")

	testError := errors.New("database connection failed")

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, testError)

	if status := responseRecorder.Code; status != http.StatusInternalServerError {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusInternalServerError,
		)
	}

	expected := `{"code":"UNEXPECTED","message":"database connection failed"}`
	if strings.TrimSpace(responseRecorder.Body.String()) != expected {
		test.Errorf(
			"HandleAPIError returned unexpected body: got %v want %v",
			responseRecorder.Body.String(),
			expected,
		)
	}
}

func TestHandleAPIError_WithRegularError_ProductionMode(test *testing.T) {
	os.Setenv("MODE", "production")
	defer os.Unsetenv("MODE")

	testError := errors.New("database connection failed")

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, testError)

	if status := responseRecorder.Code; status != http.StatusInternalServerError {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusInternalServerError,
		)
	}

	expected := `{"code":"UNEXPECTED","message":"database connection failed"}`
	if strings.TrimSpace(responseRecorder.Body.String()) != expected {
		test.Errorf(
			"HandleAPIError returned unexpected body: got %v want %v",
			responseRecorder.Body.String(),
			expected,
		)
	}
}

func TestHandleAPIError_WithNilError(test *testing.T) {
	os.Setenv("MODE", "development")
	defer os.Unsetenv("MODE")

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, nil)

	if status := responseRecorder.Code; status != http.StatusOK {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusOK,
		)
	}

	if responseRecorder.Body.String() != "" {
		test.Errorf(
			"HandleAPIError returned unexpected body: got %v want empty",
			responseRecorder.Body.String(),
		)
	}
}

func TestHandleAPIError_WithAPIError_NoMessage(test *testing.T) {
	os.Setenv("MODE", "development")
	defer os.Unsetenv("MODE")

	testError := &schemas.APIError{
		Code:  string(schemas.ErrUnexpected),
		Stack: errors.New("test stack error"),
	}

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, testError)

	if status := responseRecorder.Code; status != http.StatusInternalServerError {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusInternalServerError,
		)
	}

	expected := `{"code":"UNEXPECTED"}`
	if strings.TrimSpace(responseRecorder.Body.String()) != expected {
		test.Errorf(
			"HandleAPIError returned unexpected body: got %v want %v",
			responseRecorder.Body.String(),
			expected,
		)
	}
}

func TestHandleAPIError_WithAPIError_NoStack(test *testing.T) {
	os.Setenv("MODE", "development")
	defer os.Unsetenv("MODE")

	testError := &schemas.APIError{
		Code:    string(schemas.ErrInvalidRequest),
		Message: "Invalid request data",
	}

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, testError)

	if status := responseRecorder.Code; status != http.StatusBadRequest {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusBadRequest,
		)
	}

	expected := `{"code":"INVALID_REQUEST","message":"Invalid request data"}`
	if strings.TrimSpace(responseRecorder.Body.String()) != expected {
		test.Errorf(
			"HandleAPIError returned unexpected body: got %v want %v",
			responseRecorder.Body.String(),
			expected,
		)
	}
}

func TestHandleAPIError_WithDifferentErrorCodes(test *testing.T) {
	os.Setenv("MODE", "development")
	defer os.Unsetenv("MODE")

	testCases := []struct {
		name           string
		errorCode      schemas.APIErrorCode
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Account Not Found",
			errorCode:      schemas.ErrAccountNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"code":"ACCOUNT_NOT_FOUND","message":"Account not found"}`,
		},
		{
			name:           "Account Already Exists",
			errorCode:      schemas.ErrAccountAlreadyExists,
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"code":"ACCOUNT_ALREADY_EXISTS","message":"Account already exists"}`,
		},
		{
			name:           "Unprocessable Entity",
			errorCode:      schemas.ErrUnprocessableEntity,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   `{"code":"UNPROCESSABLE_ENTITY","message":"Unprocessable entity"}`,
		},
		{
			name:           "Database Error",
			errorCode:      schemas.ErrDatabase,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"code":"DATABASE_FAULT","message":"Database error"}`,
		},
	}

	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			testError := schemas.NewAPIError(
				testCase.errorCode,
				strings.Split(
					testCase.expectedBody,
					`"message":"`,
				)[1][:strings.Index(
					strings.Split(testCase.expectedBody, `"message":"`)[1],
					`"`,
				)],
				nil,
			)

			responseRecorder := httptest.NewRecorder()

			HandleAPIError(responseRecorder, testError)

			if status := responseRecorder.Code; status != testCase.expectedStatus {
				test.Errorf(
					"HandleAPIError returned wrong status code: got %v want %v",
					status,
					testCase.expectedStatus,
				)
			}

			if strings.TrimSpace(responseRecorder.Body.String()) != testCase.expectedBody {
				test.Errorf(
					"HandleAPIError returned unexpected body: got %v want %v",
					responseRecorder.Body.String(),
					testCase.expectedBody,
				)
			}
		})
	}
}

func TestHandleAPIError_EnvironmentVariableHandling(test *testing.T) {
	os.Unsetenv("MODE")

	testError := schemas.NewAPIError(schemas.ErrInvalidCredentials, "Test error", nil)

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, testError)

	if status := responseRecorder.Code; status != http.StatusUnauthorized {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusUnauthorized,
		)
	}
}

func TestHandleAPIError_WithComplexError(test *testing.T) {
	os.Setenv("MODE", "development")
	defer os.Unsetenv("MODE")

	complexMessage := `{"error": "complex error with "quotes" and \n newlines"}`
	testError := schemas.NewAPIError(
		schemas.ErrUnexpected,
		complexMessage,
		errors.New("stack trace"),
	)

	responseRecorder := httptest.NewRecorder()

	HandleAPIError(responseRecorder, testError)

	if status := responseRecorder.Code; status != http.StatusInternalServerError {
		test.Errorf(
			"HandleAPIError returned wrong status code: got %v want %v",
			status,
			http.StatusInternalServerError,
		)
	}

	expectedMessage := `"{\"error\": \"complex error with \"quotes\" and \\n newlines\"}"`
	if !strings.Contains(responseRecorder.Body.String(), expectedMessage) {
		test.Errorf(
			"HandleAPIError response should contain the complex message: %v",
			responseRecorder.Body.String(),
		)
	}
}

func BenchmarkHandleAPIError(benchmark *testing.B) {
	os.Setenv("MODE", "development")
	defer os.Unsetenv("MODE")

	testError := schemas.NewAPIError(
		schemas.ErrInvalidCredentials,
		"Benchmark test error",
		nil,
	)

	benchmark.ResetTimer()
	for i := 0; i < benchmark.N; i++ {
		responseRecorder := httptest.NewRecorder()
		HandleAPIError(responseRecorder, testError)
	}
}
