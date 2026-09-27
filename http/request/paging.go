package request

import (
	"fmt"
	"net/http"
	"strings"
)

// DefaultPageRequestLimit is the page size applied when the client omits
// limit, or sends limit=0.
var DefaultPageRequestLimit = 20

// MaxLimit caps the page size a client can ask for.
var MaxLimit = 200

// LimitOffset is the storage-layer paging primitive produced by the paging
// decoders. Storage layers accept it directly, regardless of how the handler
// decoded the request.
type LimitOffset struct {
	Limit  uint
	Offset uint
}

// Order is a sort-direction token ("asc"/"desc"). Storage layers interpret
// it; the decoders only validate the shape.
type Order string

// QueryVal renders the order for SQL, uppercased.
func (o Order) QueryVal() string {
	return strings.ToUpper(string(o))
}

// PagingRequest decodes limit/offset paging.
//
//	GET /items?limit=20&offset=40
type PagingRequest struct {
	Limit  int `json:"limit,omitempty" validate:"min=0,max=100"`
	Offset int `json:"offset,omitempty" validate:"min=0,max=10000"`
}

// ToPagingRequest decodes limit/offset from the query string, applying
// DefaultPageRequestLimit when limit is absent or zero.
func ToPagingRequest(r *http.Request) (PagingRequest, error) {
	limit := QueryParamIntDefault(r, ParamLimit, DefaultPageRequestLimit)
	offset := QueryParamIntDefault(r, ParamOffset, 0)
	return NewPagingRequest(limit, offset)
}

// NewPagingRequest builds a PagingRequest from raw values, validating the
// limit against MaxLimit and defaulting zero to DefaultPageRequestLimit.
func NewPagingRequest(limit, offset int) (PagingRequest, error) {
	if limit > MaxLimit || limit < 0 {
		return PagingRequest{}, NewBadRequestError(fmt.Sprintf("limit must be between 1 and %d", MaxLimit))
	}
	if limit == 0 {
		limit = DefaultPageRequestLimit
	}
	return PagingRequest{Limit: limit, Offset: offset}, nil
}

// Domain returns the storage-layer paging primitive.
func (p PagingRequest) Domain() LimitOffset {
	return LimitOffset{Limit: uint(p.Limit), Offset: uint(p.Offset)}
}
