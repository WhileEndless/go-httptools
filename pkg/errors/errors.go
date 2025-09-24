package errors

import "fmt"

// ErrorType represents different types of parsing errors
type ErrorType int

const (
	ErrorTypeInvalidFormat ErrorType = iota
	ErrorTypeMalformedHeader
	ErrorTypeInvalidMethod
	ErrorTypeInvalidURL
	ErrorTypeInvalidVersion
	ErrorTypeInvalidStatusCode
	ErrorTypeCompressionError
)

// Error represents a structured HTTP parsing error
type Error struct {
	Type    ErrorType
	Message string
	Context string
	Raw     []byte
}

func (e *Error) Error() string {
	return fmt.Sprintf("httptools: %s (context: %s)", e.Message, e.Context)
}

// NewError creates a new Error
func NewError(errType ErrorType, message, context string, raw []byte) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Context: context,
		Raw:     raw,
	}
}

// IsParseError checks if an error is a parsing error
func IsParseError(err error) bool {
	_, ok := err.(*Error)
	return ok
}
