package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	logger = NewCustomLogger(os.Stdout)
)

const (
	enableDebugLevel = "FSAS_DEBUG"
)

func NewCustomLogger(w io.Writer) *slog.Logger {
	debugLevelEnabled := os.Getenv(enableDebugLevel)
	var loggerLevel = new(slog.LevelVar)
	if strings.ToLower(debugLevelEnabled) == "true" {
		loggerLevel.Set(slog.LevelDebug)
	} else {
		loggerLevel.Set(slog.LevelInfo)
	}

	stdoutHandler := NewCustomHandler(w, &slog.HandlerOptions{Level: loggerLevel, AddSource: true})
	logger := slog.New(stdoutHandler)
	return logger
}

// ---------------------
// CustomHandler
// ---------------------

// CustomHandler implements the slog.Handler interface
type CustomHandler struct {
	writer io.Writer
	opts   *slog.HandlerOptions
}

// NewCustomHandler creates a new instance of CustomHandler
func NewCustomHandler(writer io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return &CustomHandler{writer: writer, opts: opts}
}

// Handle writes log records in the desired format
func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	timestamp := r.Time.Format("2006-01-02T15:04:05.000Z07:00")
	level := r.Level.String()
	message := r.Message
	var logLine string

	if !h.opts.AddSource {
		logLine = fmt.Sprintf("%s; [%s]; %s", timestamp, level, message)
	} else {
		fileName, lineNumber := getLogCallInfo()
		dataFromAllAttributes := getDataFromAllAttributes(r)
		dataFromAllAttributes = CensorTextWithRegex(dataFromAllAttributes)
		message = CensorTextWithRegex(message)
		if dataFromAllAttributes == "" {
			message = fmt.Sprintf("%s", message)
		} else {
			message = fmt.Sprintf("%s %s", message, dataFromAllAttributes)
		}

		logLine = fmt.Sprintf("%s:%d; %s; [%s]; %s;",
			fileName, lineNumber, timestamp, level, message)
	}

	_, err := h.writer.Write([]byte(fmt.Sprintf("%s \n", logLine)))

	if err != nil {
		return fmt.Errorf("failed to write log: %v", err)
	}
	return nil
}

// getLogCallInfo Get info from place where logger were called like:
// file name, line number
func getLogCallInfo() (fileName string, lineNumber int) {
	// The argument in function "runtime.Caller" is the number of stack frames
	// to ascend, with 0 identifying the caller of Caller. The return values report the
	// program counter and line number within the file of the corresponding call.
	_, filePath, lineNum, ok := runtime.Caller(5)
	if ok {
		fileName = filepath.Base(filePath)
		lineNumber = lineNum
	} else {
		fileName = "unknown"
		lineNumber = 0
	}
	return fileName, lineNum
}

// Function to extract all attributes from a slog.Record
func getDataFromAllAttributes(r slog.Record) string {
	var attributes []string

	// Use the Attrs method to iterate over attributes
	r.Attrs(func(attr slog.Attr) bool {
		// Format each attribute as "key=value" and append to the slice
		attributes = append(attributes, fmt.Sprintf("%s=%v", attr.Key, attr.Value.Any()))
		return true // Continue iterating
	})

	// Join all attributes with a comma and space
	return strings.Join(attributes, ", ")
}

// Enabled checks if a log level is enabled
func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.opts == nil || h.opts.Level == nil {
		return true // if no level set, enable all
	}
	return level >= h.opts.Level.Level()
}

// WithAttrs returns a new handler with additional attributes (not used here)
func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup returns a new handler with a specified group (not used here)
func (h *CustomHandler) WithGroup(name string) slog.Handler {
	return h
}

func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}
