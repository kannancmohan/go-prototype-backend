package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	log_impl "github.com/kannancmohan/go-prototype-backend/cmd/internal/common/log"
	"github.com/kannancmohan/go-prototype-backend/internal/common/log"
)

func TestSlogLogger(t *testing.T) {
	tests := []struct {
		name          string
		loggerMethod  func(logger log.Logger, msg string, args ...any)
		inputMsg      string
		inputArgs     []any
		expectedMsg   string
		expectedKey   string
		expectedValue string
	}{
		{
			name:          "Test Info",
			loggerMethod:  (log.Logger).Info,
			inputMsg:      "test info message",
			inputArgs:     []any{"info-key", "info-value"},
			expectedMsg:   "test info message",
			expectedKey:   "info-key",
			expectedValue: "info-value",
		},
		{
			name:          "Test Error",
			loggerMethod:  (log.Logger).Error,
			inputMsg:      "test error message",
			inputArgs:     []any{"error-key", "error-value"},
			expectedMsg:   "test error message",
			expectedKey:   "error-key",
			expectedValue: "error-value",
		},
		{
			name:          "Test Debug",
			loggerMethod:  (log.Logger).Debug,
			inputMsg:      "test debug message",
			inputArgs:     []any{"debug-key", "debug-value"},
			expectedMsg:   "test debug message",
			expectedKey:   "debug-key",
			expectedValue: "debug-value",
		},
		{
			name:          "Test Warn",
			loggerMethod:  (log.Logger).Warn,
			inputMsg:      "test warn message",
			inputArgs:     []any{"warn-key", "warn-value"},
			expectedMsg:   "test warn message",
			expectedKey:   "warn-key",
			expectedValue: "warn-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := log_impl.NewSlogLogger(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
			tt.loggerMethod(logger, tt.inputMsg, tt.inputArgs...)

			var logOutput map[string]any
			if err := json.Unmarshal(buf.Bytes(), &logOutput); err != nil {
				t.Fatalf("failed to unmarshal log output: %v", err)
			}

			// Verify the log message
			if logOutput["msg"] != tt.expectedMsg {
				t.Errorf("expected message %q, got %q", tt.expectedMsg, logOutput["msg"])
			}

			// Verify the log key-value pair
			if logOutput[tt.expectedKey] != tt.expectedValue {
				t.Errorf("expected %q=%q, got %q=%q", tt.expectedKey, tt.expectedValue, tt.expectedKey, logOutput[tt.expectedKey])
			}
		})
	}
}

func TestSlogLoggerWithContext(t *testing.T) {
	// Define test cases
	tests := []struct {
		name                 string
		loggerMethod         func(logger log.Logger, msg string, args ...any)
		inputMsg             string
		inputArgs            []any
		inputContextArgs     []any
		expectedMsg          string
		expectedKey          string
		expectedValue        string
		expectedContextKey   string
		expectedContextValue string
	}{
		{
			name:                 "Test info log using WithContext",
			loggerMethod:         (log.Logger).Info,
			inputMsg:             "test info message with context",
			inputArgs:            []any{"info-key", "info-value"},
			inputContextArgs:     []any{testContextKey, "info-ctx-value"},
			expectedMsg:          "test info message with context",
			expectedKey:          "info-key",
			expectedValue:        "info-value",
			expectedContextKey:   string(testContextKey),
			expectedContextValue: "info-ctx-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			ctx := context.WithValue(context.Background(), tt.inputContextArgs[0], tt.inputContextArgs[1])
			slogHandler := newTestSlogHandler(string(testContextKey), slog.NewJSONHandler(&buf, nil))
			logger := log_impl.NewSlogLogger(slog.New(slogHandler))
			loggerWithContext := logger.WithContext(ctx)

			tt.loggerMethod(loggerWithContext, tt.inputMsg, tt.inputArgs...)

			var logOutput map[string]any
			if err := json.Unmarshal(buf.Bytes(), &logOutput); err != nil {
				t.Fatalf("failed to unmarshal log output: %v", err)
			}

			// Verify the log message
			if logOutput["msg"] != tt.expectedMsg {
				t.Errorf("expected message %q, got %q", tt.expectedMsg, logOutput["msg"])
			}

			// Verify the log key-value pair
			if logOutput[tt.expectedKey] != tt.expectedValue {
				t.Errorf("expected %q=%q, got %q=%q", tt.expectedKey, tt.expectedValue, tt.expectedKey, logOutput[tt.expectedKey])
			}

			// Verify the context key-value pair
			if logOutput[tt.expectedContextKey] != tt.expectedContextValue {
				t.Errorf("expected context %q=%q, got %q=%q", tt.expectedContextKey, tt.expectedContextValue, tt.expectedContextKey, logOutput[tt.expectedContextKey])
			}
		})
	}
}

func TestSlogLoggerWith(t *testing.T) {
	// Define test cases
	tests := []struct {
		name              string
		loggerMethod      func(logger log.Logger, msg string, args ...any)
		inputMsg          string
		inputArgs         []any
		inputWithArgs     []any
		expectedMsg       string
		expectedKey       string
		expectedValue     string
		expectedWithKey   string
		expectedWithValue string
	}{
		{
			name:              "Test info log using with",
			loggerMethod:      (log.Logger).Info,
			inputMsg:          "test info message with context",
			inputArgs:         []any{"info-key", "info-value"},
			inputWithArgs:     []any{"info-addition-key", "info-addition-value"},
			expectedMsg:       "test info message with context",
			expectedKey:       "info-key",
			expectedValue:     "info-value",
			expectedWithKey:   "info-addition-key",
			expectedWithValue: "info-addition-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			logger := log_impl.NewSlogLogger(slog.New(slog.NewJSONHandler(&buf, nil)))
			loggerWithFields := logger.With(tt.inputWithArgs...)

			tt.loggerMethod(loggerWithFields, tt.inputMsg, tt.inputArgs...)

			var logOutput map[string]any
			if err := json.Unmarshal(buf.Bytes(), &logOutput); err != nil {
				t.Fatalf("failed to unmarshal log output: %v", err)
			}

			// Verify the log message
			if logOutput["msg"] != tt.expectedMsg {
				t.Errorf("expected message %q, got %q", tt.expectedMsg, logOutput["msg"])
			}

			// Verify the log key-value pair
			if logOutput[tt.expectedKey] != tt.expectedValue {
				t.Errorf("expected %q=%q, got %q=%q", tt.expectedKey, tt.expectedValue, tt.expectedKey, logOutput[tt.expectedKey])
			}

			// Verify the additional key-value pair added by With
			if logOutput[tt.expectedWithKey] != tt.expectedWithValue {
				t.Errorf("expected additional field %q=%q, got %q=%q", tt.expectedWithKey, tt.expectedWithValue, tt.expectedWithKey, logOutput[tt.expectedWithKey])
			}
		})
	}
}

type contextKey string

const testContextKey contextKey = "testContextKey1"

// testSlogHandler. custom slog handler to include value from context
type testSlogHandler struct {
	logKey string
	next   slog.Handler
}

func newTestSlogHandler(logKey string, next slog.Handler) *testSlogHandler {
	return &testSlogHandler{logKey: logKey, next: next}
}

func (r *testSlogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return r.next.Enabled(ctx, lvl)
}

func (r *testSlogHandler) Handle(ctx context.Context, rec slog.Record) error {
	val, ok := ctx.Value(testContextKey).(string)
	if ok {
		rec.AddAttrs(slog.Any(r.logKey, val))
	}
	return r.next.Handle(ctx, rec)
}

func (r *testSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &testSlogHandler{
		next: r.next.WithAttrs(attrs),
	}
}

func (r *testSlogHandler) WithGroup(name string) slog.Handler {
	return &testSlogHandler{
		next: r.next.WithGroup(name),
	}
}
