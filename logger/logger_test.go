package logger

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func captureLogOutput(f func(_ string, _ ...any), msg string, args ...any) string {
	// 1. save original logger
	originalSlogLogger := slog.Default()
	defer func() {
		slog.SetDefault(originalSlogLogger)
	}()

	// 2. Create a pipe for capturing output
	r, w, _ := os.Pipe()

	// 3. Create a new slog.Logger that writes to the pipe's write end
	logger = NewCustomLogger(w)

	// Create a channel to signal when the goroutine has finished reading
	done := make(chan struct{})
	var capturedOutput bytes.Buffer

	// Start a goroutine to read from the pipe's read end
	go func() {
		defer close(done)
		_, err := io.Copy(&capturedOutput, r)
		if err != nil {
			fmt.Printf("Error reading from pipe: %v\n", err)
		}
	}()

	// 5. Call the function that uses the logger
	f(msg, args...)

	// Close the write end of the pipe to signal EOF to the reader goroutine
	w.Close()

	// Wait for the goroutine to finish reading
	<-done

	// 6. Get the captured output as a string
	return capturedOutput.String()
}

func Test_verifyLogFormat(t *testing.T) {
	err := os.Setenv(enableDebugLevel, "true")
	if err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}

	defer func() {
		err := os.Setenv(enableDebugLevel, "")
		if err != nil {
			t.Fatalf("failed to set env var: %v", err)
		}
	}()

	testCases := []struct {
		name          string
		logLevel      func(msg string, args ...any)
		message       string
		attributes    []any
		expectedPart2 string
	}{
		{name: "level Debug, no attributes",
			logLevel:      Debug,
			message:       "Hello world!",
			attributes:    nil,
			expectedPart2: "[DEBUG]; Hello world!",
		},
		{name: "level Debug, with attributes",
			logLevel:      Debug,
			message:       "Hello world!",
			attributes:    []any{"foo", 11},
			expectedPart2: "[DEBUG]; Hello world! foo=11;",
		},

		{name: "level Info, no attributes",
			logLevel:      Info,
			message:       "Hello world!",
			attributes:    nil,
			expectedPart2: "[INFO]; Hello world!",
		},
		{name: "level Info, with attributes",
			logLevel:      Info,
			message:       "Hello world!",
			attributes:    []any{"foo", 11},
			expectedPart2: "[INFO]; Hello world! foo=11;",
		},

		{name: "level Warn, no attributes",
			logLevel:      Warn,
			message:       "Hello world!",
			attributes:    nil,
			expectedPart2: "[WARN]; Hello world!",
		},
		{name: "level Warn, with attributes",
			logLevel:      Warn,
			message:       "Hello world!",
			attributes:    []any{"foo", 11},
			expectedPart2: "[WARN]; Hello world! foo=11;",
		},

		{name: "level Error, no attributes",
			logLevel:      Error,
			message:       "Hello world!",
			attributes:    nil,
			expectedPart2: "[ERROR]; Hello world!",
		},
		{name: "level Error, with attributes",
			logLevel:      Error,
			message:       "Hello world!",
			attributes:    []any{"foo", 11},
			expectedPart2: "[ERROR]; Hello world! foo=11;",
		},
		{name: "level Error, with 4 attributes",
			logLevel:      Error,
			message:       "Hello world!",
			attributes:    []any{"foo", 11, "pi", 3.14},
			expectedPart2: "[ERROR]; Hello world! foo=11, pi=3.14",
		},

		{name: "level Info, with censored sensitive data",
			logLevel:      Info,
			message:       `password=supersecret&secret=topsecret&"access_token":"abc123"&"refresh_token":"r123"&"id_token":"i123"`,
			attributes:    nil,
			expectedPart2: `[INFO]; password=[REDACTED]&secret=[REDACTED]&"access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED];`,
		},
		{name: "level Info, with censored sensitive data in attributes",
			logLevel:      Info,
			message:       "hello world:",
			attributes:    []any{"foo", "password=supersecret&secret=topsecret&", "bar", `"access_token":"abc123"&"refresh_token":"r123"&"id_token":"i123"`},
			expectedPart2: `[INFO]; hello world: foo=password=[REDACTED]&secret=[REDACTED]&, bar="access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED];`,
		},

		{name: "level Debug, with censored sensitive data in attributes (real life example)",
			logLevel: Debug,
			message:  "Initiating POST request:",
			attributes: []any{
				"endpoint", "/realms/12345678-1234-1234-1234-123456789012/protocol/openid-connect/token/introspect",
				"payload", "client_id=cdi&client_secret=topsecret&token=eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJmT1VSOXpEcnZ2MFpnaEx2TUJPcEUzTT"},
			expectedPart2: `[DEBUG]; Initiating POST request: endpoint=/realms/12345678-1234-1234-1234-123456789012/protocol/openid-connect/token/introspect, payload=client_id=cdi&client_secret=[REDACTED]&token=[REDACTED]`,
		},

		{name: "level Warn, with censored sensitive data; real-life http request with default phrases",
			logLevel:      Warn,
			message:       "Initiating POST request: ; endpoint=/realms/12345678-1234-1234-1234-123456789012/protocol/openid-connect/token, payload=client_id=cdi&client_secret=SensitiveInfo&grant_type=password&password=foobar&response=id_token+token&scope=openid&username=jdoe;",
			attributes:    nil,
			expectedPart2: "[WARN]; Initiating POST request: ; endpoint=/realms/12345678-1234-1234-1234-123456789012/protocol/openid-connect/token, payload=client_id=cdi&client_secret=[REDACTED]&grant_type=password&password=[REDACTED]&response=id_token+token&scope=openid&username=jdoe;",
		},

		{name: "level Error, with censored sensitive data; simplified real-life http response with default phrases",
			logLevel:      Error,
			message:       `response_body={"access_token":"eyJhn0.eyJQ.W418g","expires_in":1750,"refresh_expires_in":7200,"refresh_token":"eyJhbGcX43A","token_type":"Bearer","id_token":"eyJhbGciOi36vyHeg","not-before-policy":0,"session_state":"d8321164-bd12-4606-922e-f170f2b2088d","scope":"openid pgcdi_privileges email profile"};`,
			attributes:    nil,
			expectedPart2: `[ERROR]; response_body={"access_token":[REDACTED],"expires_in":1750,"refresh_expires_in":7200,"refresh_token":[REDACTED],"token_type":"Bearer","id_token":[REDACTED],"not-before-policy":0,"session_state":"d8321164-bd12-4606-922e-f170f2b2088d","scope":"openid pgcdi_privileges email profile"};`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			timestamp := time.Now().Format("2006-01-02T15:04:05")
			expectedMessagePart1 := fmt.Sprintf("logger_test.go:42; %s", timestamp)

			output := captureLogOutput(tc.logLevel, tc.message, tc.attributes...)

			assert.Contains(t, output, expectedMessagePart1, fmt.Sprintf("expected output to contain \n%q but got \n%q", expectedMessagePart1, output))
			assert.Contains(t, output, tc.expectedPart2, fmt.Sprintf("expected output to contain \n%q but got \n%q", tc.expectedPart2, output))
		})
	}

}

func Test_enableDisableDebugLevel(t *testing.T) {

	output := captureLogOutput(Debug, "Hello world", nil)

	assert.Equal(t, output, "", fmt.Sprintf("expected output should be empty but got %q", output))

	err := os.Setenv(enableDebugLevel, "true")
	if err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}

	defer func() {
		err := os.Setenv(enableDebugLevel, "")
		if err != nil {
			t.Fatalf("failed to set env var: %v", err)
		}
	}()

	output = captureLogOutput(Debug, "Hello world", nil)
	expected := "Hello world"

	assert.Contains(t, output, expected, fmt.Sprintf("expected output to contain %q but got %q", expected, output))
}
