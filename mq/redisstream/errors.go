package redisstream

import "errors"

var (
	ErrMissingAddr = errors.New("redisstream: addr is required")

	ErrMissingStreamName = errors.New("redisstream: stream name is required")

	ErrMissingGroup = errors.New("redisstream: group is required")

	// ErrBlockTimeoutTooLong is returned when the consumer would block on
	// XREADGROUP for longer than the client's own read timeout, which would
	// surface as a spurious i/o timeout on every idle poll.
	ErrBlockTimeoutTooLong = errors.New("redisstream: block_timeout must be shorter than the client read timeout")
)
