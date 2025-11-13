package rawhttp

import (
	"errors"
	"fmt"
)

// Error types for different failure scenarios
var (
	ErrDNSResolution    = errors.New("DNS resolution failed")
	ErrConnection       = errors.New("connection failed")
	ErrTLSHandshake     = errors.New("TLS handshake failed")
	ErrTimeout          = errors.New("operation timeout")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrProxyConnection  = errors.New("proxy connection failed")
	ErrBodyTooLarge     = errors.New("response body too large")
	ErrProtocolNegotiation = errors.New("protocol negotiation failed")
)

// ErrorType represents different error categories
type ErrorType int

const (
	ErrorTypeDNS ErrorType = iota
	ErrorTypeConnection
	ErrorTypeTLS
	ErrorTypeTimeout
	ErrorTypeProtocol
	ErrorTypeProxy
)

// HTTPError represents a detailed HTTP error with categorization
type HTTPError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

// NewDNSError creates a DNS resolution error
func NewDNSError(err error) *HTTPError {
	return &HTTPError{
		Type:    ErrorTypeDNS,
		Message: "DNS resolution failed",
		Err:     err,
	}
}

// NewConnectionError creates a connection error
func NewConnectionError(err error) *HTTPError {
	return &HTTPError{
		Type:    ErrorTypeConnection,
		Message: "connection failed",
		Err:     err,
	}
}

// NewTLSError creates a TLS handshake error
func NewTLSError(err error) *HTTPError {
	return &HTTPError{
		Type:    ErrorTypeTLS,
		Message: "TLS handshake failed",
		Err:     err,
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(err error) *HTTPError {
	return &HTTPError{
		Type:    ErrorTypeTimeout,
		Message: "operation timeout",
		Err:     err,
	}
}

// NewProxyError creates a proxy connection error
func NewProxyError(err error) *HTTPError {
	return &HTTPError{
		Type:    ErrorTypeProxy,
		Message: "proxy connection failed",
		Err:     err,
	}
}

// NewProtocolError creates a protocol negotiation error
func NewProtocolError(err error) *HTTPError {
	return &HTTPError{
		Type:    ErrorTypeProtocol,
		Message: "protocol negotiation failed",
		Err:     err,
	}
}
