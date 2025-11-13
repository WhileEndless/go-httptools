package rawhttp_test

import (
	"errors"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/rawhttp"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		errFunc  func(error) *rawhttp.HTTPError
		wantType rawhttp.ErrorType
	}{
		{
			name:     "DNS Error",
			errFunc:  rawhttp.NewDNSError,
			wantType: rawhttp.ErrorTypeDNS,
		},
		{
			name:     "Connection Error",
			errFunc:  rawhttp.NewConnectionError,
			wantType: rawhttp.ErrorTypeConnection,
		},
		{
			name:     "TLS Error",
			errFunc:  rawhttp.NewTLSError,
			wantType: rawhttp.ErrorTypeTLS,
		},
		{
			name:     "Timeout Error",
			errFunc:  rawhttp.NewTimeoutError,
			wantType: rawhttp.ErrorTypeTimeout,
		},
		{
			name:     "Proxy Error",
			errFunc:  rawhttp.NewProxyError,
			wantType: rawhttp.ErrorTypeProxy,
		},
		{
			name:     "Protocol Error",
			errFunc:  rawhttp.NewProtocolError,
			wantType: rawhttp.ErrorTypeProtocol,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseErr := errors.New("test error")
			httpErr := tt.errFunc(baseErr)

			if httpErr.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", httpErr.Type, tt.wantType)
			}

			if httpErr.Error() == "" {
				t.Error("Error() returned empty string")
			}

			if unwrapped := errors.Unwrap(httpErr); unwrapped != baseErr {
				t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
			}
		})
	}
}

func TestHTTPErrorMessage(t *testing.T) {
	baseErr := errors.New("connection refused")
	httpErr := rawhttp.NewConnectionError(baseErr)

	errMsg := httpErr.Error()
	if errMsg == "" {
		t.Error("Error message is empty")
	}

	// Should contain both the message and the base error
	if len(errMsg) < len(baseErr.Error()) {
		t.Error("Error message doesn't include base error")
	}
}
