package redisstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamKey_IsNamespaced(t *testing.T) {
	assert.Equal(t, "ns:orders", streamKey("ns", "orders"))
	assert.NotEqual(t, streamKey("ns", "orders"), streamKey("other", "orders"))
}
