package log

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newBufLogger builds a logger writing info+ to buf and error+ to errBuf.
func newBufLogger(buf, errBuf *bytes.Buffer) *zap.SugaredLogger {
	return New(
		WithLevel(zapcore.DebugLevel),
		WithOutput(zapcore.AddSync(buf)),
		WithErrorOutput(zapcore.AddSync(errBuf)),
	)
}

func TestNew_WithErrorOutputRoutesErrors(t *testing.T) {
	var buf, errBuf bytes.Buffer
	logger := newBufLogger(&buf, &errBuf)

	logger.Info("hello")
	logger.Error("bad")

	assert.Contains(t, buf.String(), "hello")
	assert.Contains(t, buf.String(), "bad", "error entries also reach the main output")
	assert.Contains(t, errBuf.String(), "bad")
	assert.NotContains(t, errBuf.String(), "hello")
}

func TestNew_WithTimeFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithOutput(zapcore.AddSync(&buf)),
		WithTimeFormat("2006"),
		WithLevel(zapcore.DebugLevel),
	)

	logger.Info("ts check")

	require.Contains(t, buf.String(), "ts check")
	// The custom layout renders only the 4-digit year; the default ISO8601
	// format would have rendered a full timestamp.
	assert.NotContains(t, buf.String(), "-")
}
