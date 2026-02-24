package log

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew(t *testing.T) {
	logger := New()
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestNewWithLevel(t *testing.T) {
	tests := []struct {
		name  string
		level zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(WithLevel(tt.level))
			if logger == nil {
				t.Fatal("New() returned nil logger")
			}
		})
	}
}

func TestNewWithLevelString(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(WithLevelString(tt.level))
			if logger == nil {
				t.Fatal("New() returned nil logger")
			}
		})
	}
}

func TestNewWithEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
	}{
		{"console", "console"},
		{"json", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(WithEncoding(tt.encoding))
			if logger == nil {
				t.Fatal("New() returned nil logger")
			}
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
			tt.logFn("test message")
			tt.logFfn("test %s", "message")
			tt.logWfn("test", "key", "value")
		})
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Fatal("Default() returned nil logger")
	}
}

func TestSetDefault(t *testing.T) {
	original := Default()
	defer SetDefault(original)

	newLogger := zap.NewNop().Sugar()
	SetDefault(newLogger)

	if Default() == original {
		t.Error("SetDefault() did not change the default logger")
	}
}

func TestWithOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(zapcore.AddSync(&buf)))
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}

	logger.Info("test message")
	if !strings.Contains(buf.String(), "test message") {
		t.Error("Expected log message in output")
	}
}

func TestWithDevelopment(t *testing.T) {
	logger := New(WithDevelopment(true))
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestSync(t *testing.T) {
	err := Sync()
	// Sync may fail on some buffers (e.g., stdout to closed pipe)
	// but should not panic
	_ = err
}
