package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
)

// TrimStrings middleware automatically trims whitespace from string fields
// in JSON body strings before they reach the handler.
//
// This ensures that all string inputs are cleaned of leading/trailing whitespace
// without requiring manual trimming in each handler or service.
func TrimStrings(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil || r.Body == http.NoBody {
			next.ServeHTTP(w, r)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			// Pass through whatever was read; the handler's decoder decides.
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}

		if len(bodyBytes) == 0 {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}

		var data map[string]any
		if err := sonic.Unmarshal(bodyBytes, &data); err != nil {
			// Invalid JSON — let the handler's decoder answer.
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}
		trimStringsRecursive(data)

		encoded, err := sonic.Marshal(data)
		if err != nil {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(encoded))
		next.ServeHTTP(w, r)
	})
}

// trimStringsRecursive recursively trims all string values in the data structure
func trimStringsRecursive(v any) {
	switch val := v.(type) {
	case map[string]any:
		for k, item := range val {
			if str, ok := item.(string); ok {
				val[k] = trimString(str)
			} else {
				trimStringsRecursive(item)
			}
		}
	case []any:
		for i, item := range val {
			if str, ok := item.(string); ok {
				val[i] = trimString(str)
			} else {
				trimStringsRecursive(item)
			}
		}
	}
}

// trimString trims leading and trailing whitespace from a string
func trimString(s string) string {
	if s == "" {
		return s
	}
	if s[0] != ' ' && s[len(s)-1] != ' ' {
		return s
	}

	start := 0
	end := len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}
