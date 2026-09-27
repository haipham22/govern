package request

import (
	"encoding"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

// PagingCursorRequest decodes limit/cursor/order paging. Cursor is generic:
// pass a *T implementing encoding.TextUnmarshaler — e.g. *time.Time,
// *uuid.UUID, or a domain ID type — and the decoder returns a freshly
// allocated *T on valid input.
//
//	GET /items?limit=20&cursor=eyJpZCI6Li4ufQ&order=desc
type PagingCursorRequest[T encoding.TextUnmarshaler] struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor T      `json:"cursor,omitempty"`
	Order  string `json:"order,omitempty"`
}

// ToPagingCursorRequest decodes limit/cursor/order from the query string,
// applying DefaultPageRequestLimit and the "desc" default order.
func ToPagingCursorRequest[T encoding.TextUnmarshaler](r *http.Request) (PagingCursorRequest[T], error) {
	limit := QueryParamIntDefault(r, ParamLimit, DefaultPageRequestLimit)
	cursor := QueryParam(r, ParamCursor)
	order := QueryParamDefault(r, ParamOrder, "desc")
	return newPagingCursorRequest[T](limit, cursor, order)
}

func newPagingCursorRequest[T encoding.TextUnmarshaler](limit int, cursor, order string) (PagingCursorRequest[T], error) {
	if limit > MaxLimit || limit < 0 {
		return PagingCursorRequest[T]{}, NewBadRequestError(fmt.Sprintf("limit must be between 1 and %d", MaxLimit))
	}
	if limit == 0 {
		limit = DefaultPageRequestLimit
	}

	var t T
	if cursor != "" {
		tType := reflect.TypeOf(t)
		if tType.Kind() != reflect.Ptr {
			return PagingCursorRequest[T]{}, errors.New("cursor type should be pointer type instead value type")
		}
		concrete := reflect.New(tType.Elem()).Interface().(T)
		if err := concrete.UnmarshalText([]byte(cursor)); err != nil {
			return PagingCursorRequest[T]{}, NewBadRequestError(err.Error())
		}
		t = concrete
	}

	return PagingCursorRequest[T]{
		Limit:  limit,
		Cursor: t,
		Order:  order,
	}, nil
}
