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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			timestamp := time.Now().Format("2006-01-02T15:04:05")
			expectedMessagePart1 := fmt.Sprintf("logger_test.go:42; %s", timestamp)

			output := captureLogOutput(tc.logLevel, tc.message, tc.attributes...)

			assert.Contains(t, output, expectedMessagePart1, fmt.Sprintf("expected output to contain %q but got %q", expectedMessagePart1, output))
			assert.Contains(t, output, tc.expectedPart2, fmt.Sprintf("expected output to contain %q but got %q", tc.expectedPart2, output))
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
