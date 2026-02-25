package log

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Test helpers

func assertLoggerNotNil(t *testing.T, logger *zap.SugaredLogger) {
	t.Helper()
	require.NotNil(t, logger, "New() returned nil logger")
}

func TestNew(t *testing.T) {
	logger := New()
	assertLoggerNotNil(t, logger)
}

func TestNewWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		option   Option
		validate func(*testing.T, *zap.SugaredLogger)
	}{
		{
			name:   "with debug level",
			option: WithLevel(zapcore.DebugLevel),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with info level",
			option: WithLevel(zapcore.InfoLevel),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with warn level",
			option: WithLevel(zapcore.WarnLevel),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with error level",
			option: WithLevel(zapcore.ErrorLevel),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with debug level string",
			option: WithLevelString("debug"),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with info level string",
			option: WithLevelString("info"),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with warn level string",
			option: WithLevelString("warn"),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with error level string",
			option: WithLevelString("error"),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with invalid level string",
			option: WithLevelString("invalid"),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				// Should still create logger, just ignore invalid level
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with console encoding",
			option: WithEncoding("console"),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with json encoding",
			option: WithEncoding("json"),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
		{
			name:   "with development mode",
			option: WithDevelopment(true),
			validate: func(t *testing.T, logger *zap.SugaredLogger) {
				assertLoggerNotNil(t, logger)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.option)
			tt.validate(t, logger)
		})
	}
}

func TestHelpers(t *testing.T) {
	tests := []struct {
		name   string
		logFn  func(...interface{})
		logFfn func(string, ...interface{})
		logWfn func(string, ...interface{})
	}{
		{"Debug", Debug, Debugf, Debugw},
		{"Info", Info, Infof, Infow},
		{"Warn", Warn, Warnf, Warnw},
		{"Error", Error, Errorf, Errorw},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These should not panic
			assert.NotPanics(t, func() {
				tt.logFn("test message")
				tt.logFfn("test %s", "message")
				tt.logWfn("test", "key", "value")
			})
		})
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	assertLoggerNotNil(t, logger)
}

func TestSetDefault(t *testing.T) {
	original := Default()
	defer SetDefault(original)

	newLogger := zap.NewNop().Sugar()
	SetDefault(newLogger)

	assert.NotEqual(t, original, Default(), "SetDefault() did not change the default logger")
}

func TestWithOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(zapcore.AddSync(&buf)))
	assertLoggerNotNil(t, logger)

	logger.Info("test message")
	assert.Contains(t, buf.String(), "test message", "Expected log message in output")
}

func TestSync(t *testing.T) {
	err := Sync()
	// Sync may fail on some buffers (e.g., stdout to closed pipe)
	// but should not panic
	_ = err
}
