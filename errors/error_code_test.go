package errors_test

import (
	"io"
	"testing"

	"github.com/haipham22/govern/errors"
)

func TestErrorWithCode(t *testing.T) {
	tests := []struct {
		name string
		code errors.ErrorCode
		err  error
		want string
	}{
		{
			name: "with error",
			code: errors.CodeNotFound,
			err:  io.EOF,
			want: "[NOT_FOUND] EOF",
		},
		{
			name: "with nil error",
			code: errors.CodeInternal,
			err:  nil,
			want: "INTERNAL",
		},
		{
			name: "with custom message",
			code: errors.CodeInvalid,
			err:  errors.New("invalid input"),
			want: "[INVALID] invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &errors.ErrorWithCode{Code: tt.code, Err: tt.err}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorWithCode_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	e := &errors.ErrorWithCode{Code: errors.CodeInternal, Err: baseErr}

	if unwrapped := e.Unwrap(); unwrapped != baseErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
	}
}

func TestNewCode(t *testing.T) {
	err := errors.NewCode(errors.CodeNotFound, "user not found")
	if err == nil {
		t.Fatal("NewCode() returned nil")
	}

	code, ok := errors.GetCode(err)
	if !ok {
		t.Fatal("GetCode() returned false")
	}
	if code != errors.CodeNotFound {
		t.Errorf("GetCode() = %v, want %v", code, errors.CodeNotFound)
	}
}

func TestWrapCode(t *testing.T) {
	baseErr := errors.New("base error")
	err := errors.WrapCode(errors.CodeInternal, baseErr)

	if err == nil {
		t.Fatal("WrapCode() returned nil")
	}

	code, ok := errors.GetCode(err)
	if !ok {
		t.Fatal("GetCode() returned false")
	}
	if code != errors.CodeInternal {
		t.Errorf("GetCode() = %v, want %v", code, errors.CodeInternal)
	}

	// Test wrapping nil
	err = errors.WrapCode(errors.CodeInternal, nil)
	if err != nil {
		t.Errorf("WrapCode(nil) = %v, want nil", err)
	}
}

func TestGetCode(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := errors.WrapCode(errors.CodeNotFound, baseErr)

	tests := []struct {
		name     string
		err      error
		wantCode errors.ErrorCode
		wantOk   bool
	}{
		{
			name:     "with code",
			err:      wrappedErr,
			wantCode: errors.CodeNotFound,
			wantOk:   true,
		},
		{
			name:     "without code",
			err:      baseErr,
			wantCode: "",
			wantOk:   false,
		},
		{
			name:     "nil error",
			err:      nil,
			wantCode: "",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := errors.GetCode(tt.err)
			if ok != tt.wantOk {
				t.Errorf("GetCode() ok = %v, want %v", ok, tt.wantOk)
			}
			if code != tt.wantCode {
				t.Errorf("GetCode() code = %v, want %v", code, tt.wantCode)
			}
		})
	}
}

func TestIsCode(t *testing.T) {
	err := errors.WrapCode(errors.CodeNotFound, errors.New("not found"))

	tests := []struct {
		name string
		err  error
		code errors.ErrorCode
		want bool
	}{
		{
			name: "matching code",
			err:  err,
			code: errors.CodeNotFound,
			want: true,
		},
		{
			name: "different code",
			err:  err,
			code: errors.CodeInternal,
			want: false,
		},
		{
			name: "no code",
			err:  errors.New("plain error"),
			code: errors.CodeNotFound,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.IsCode(tt.err, tt.code); got != tt.want {
				t.Errorf("IsCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *errors.ErrorWithCode
		code errors.ErrorCode
	}{
		{"ErrInternal", errors.ErrInternal, errors.CodeInternal},
		{"ErrInvalid", errors.ErrInvalid, errors.CodeInvalid},
		{"ErrNotFound", errors.ErrNotFound, errors.CodeNotFound},
		{"ErrUnauthorized", errors.ErrUnauthorized, errors.CodeUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("%s.Code = %v, want %v", tt.name, tt.err.Code, tt.code)
			}
			if !errors.IsCode(tt.err, tt.code) {
				t.Errorf("%s should be identified with IsCode", tt.name)
			}
		})
	}
}
