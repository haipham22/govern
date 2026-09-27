// Package request decodes query parameters and paging requests from an
// *http.Request. Helpers are transport-framework agnostic: an Echo handler
// passes c.Request(), a chi or stdlib handler passes the request directly.
//
// Malformed input surfaces as Error, which carries the HTTP status and —
// when the offending parameter is known — its name in Field, so callers can
// render a precise client error without re-parsing anything.
package request

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Query parameter names shared by the paging decoders.
const (
	ParamLimit  = "limit"
	ParamOffset = "offset"
	ParamOrder  = "order"
	ParamCursor = "cursor"
)

// Error is a request-input failure with the HTTP status to answer with.
// Field names the offending parameter when one applies, so form-driven
// clients can mark the exact input; it is empty for errors about the whole
// request.
type Error struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// NewBadRequestError builds a 400 Error about the request as a whole.
func NewBadRequestError(message string) *Error {
	return &Error{Status: http.StatusBadRequest, Message: message}
}

// NewFieldBadRequestError builds a 400 Error pointing at one parameter.
func NewFieldBadRequestError(param, message string) *Error {
	return &Error{Status: http.StatusBadRequest, Message: message, Field: param}
}

// QueryParam returns the raw query parameter value, or "".
func QueryParam(r *http.Request, param string) string {
	return r.URL.Query().Get(param)
}

// QueryParamDefault returns the query parameter, or defaultVal when unset.
func QueryParamDefault(r *http.Request, param, defaultVal string) string {
	if raw := QueryParam(r, param); raw != "" {
		return raw
	}
	return defaultVal
}

// HasParam reports whether the query parameter is present and non-empty.
func HasParam(r *http.Request, param string) bool {
	return r.URL.Query().Get(param) != ""
}

// QueryParamRequired returns the query parameter or a 400 Error naming it.
func QueryParamRequired(r *http.Request, param string) (string, error) {
	if val := QueryParam(r, param); val != "" {
		return val, nil
	}
	return "", NewFieldBadRequestError(param, fmt.Sprintf("require param '%s'", param))
}

// QueryParams returns the comma-separated query parameter as a trimmed slice.
func QueryParams(r *http.Request, param string) []string {
	parts := strings.Split(QueryParam(r, param), ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, strings.TrimSpace(part))
	}
	return values
}

// QueryParamIntDefault returns the query parameter as an int, falling back to
// defaultVal when unset or unparseable.
func QueryParamIntDefault(r *http.Request, param string, defaultVal int) int {
	if n, err := strconv.Atoi(QueryParam(r, param)); err == nil {
		return n
	}
	return defaultVal
}

// QueryParamUint returns the query parameter as a uint64 or a 400 Error.
func QueryParamUint(r *http.Request, param string) (uint64, error) {
	raw := QueryParam(r, param)
	if raw == "" {
		return 0, NewFieldBadRequestError(param, fmt.Sprintf("missing param '%s'", param))
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, NewFieldBadRequestError(param, fmt.Sprintf("invalid format unsigned int param '%s'", raw))
	}
	return n, nil
}

// QueryParamTime returns the query parameter as a Unix-seconds timestamp or a
// 400 Error.
func QueryParamTime(r *http.Request, param string) (time.Time, error) {
	raw := QueryParam(r, param)
	if raw == "" {
		return time.Time{}, NewFieldBadRequestError(param, fmt.Sprintf("missing timestamp param '%s'", param))
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, NewFieldBadRequestError(param, fmt.Sprintf("invalid format of timestamp param '%s'", raw))
	}
	return time.Unix(sec, 0), nil
}

// QueryParamUUID returns the query parameter parsed as a UUID, or a 400 Error
// naming the parameter.
func QueryParamUUID(r *http.Request, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(QueryParam(r, param))
	if err != nil {
		return uuid.Nil, NewFieldBadRequestError(param, "must be a valid UUID")
	}
	return id, nil
}

// QueryParamUUIDOptional decodes an optional filter: absent is valid and
// yields (nil, true); malformed yields (nil, false). Callers reading several
// optional filters answer for all of them at once.
func QueryParamUUIDOptional(r *http.Request, param string) (*uuid.UUID, bool) {
	raw := QueryParam(r, param)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, false
	}
	return &id, true
}

// QueryParamUUIDs returns a comma-separated list of UUIDs, or a 400 Error
// naming the parameter. An absent param yields nil.
func QueryParamUUIDs(r *http.Request, param string) ([]uuid.UUID, error) {
	if !HasParam(r, param) {
		return nil, nil
	}
	raws := QueryParams(r, param)
	ids := make([]uuid.UUID, 0, len(raws))
	for _, raw := range raws {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, NewFieldBadRequestError(param, "must be a list of valid UUIDs")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// cursorDecodeErr marks a cursor that is not valid base64.
var cursorDecodeErr = errors.New("invalid cursor, require base64 format")
