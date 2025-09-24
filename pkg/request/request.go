package request

import (
	"fmt"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// Request represents a parsed HTTP request
type Request struct {
	Method  string                  // HTTP method (GET, POST, etc.)
	URL     string                  // Request URL/path
	Version string                  // HTTP version (HTTP/1.1, HTTP/2, etc.)
	Headers *headers.OrderedHeaders // Headers with preserved order
	Body    []byte                  // Request body
	Raw     []byte                  // Original raw request data
}

// NewRequest creates a new Request instance
func NewRequest() *Request {
	return &Request{
		Headers: headers.NewOrderedHeaders(),
	}
}

// Clone creates a deep copy of the request
func (r *Request) Clone() *Request {
	clone := NewRequest()
	clone.Method = r.Method
	clone.URL = r.URL
	clone.Version = r.Version
	clone.Body = make([]byte, len(r.Body))
	copy(clone.Body, r.Body)
	clone.Raw = make([]byte, len(r.Raw))
	copy(clone.Raw, r.Raw)

	// Clone headers
	for _, header := range r.Headers.All() {
		clone.Headers.Set(header.Name, header.Value)
	}

	return clone
}

// GetContentLength returns the Content-Length header value
func (r *Request) GetContentLength() string {
	return r.Headers.Get("Content-Length")
}

// GetContentType returns the Content-Type header value
func (r *Request) GetContentType() string {
	return r.Headers.Get("Content-Type")
}

// GetHost returns the Host header value
func (r *Request) GetHost() string {
	return r.Headers.Get("Host")
}

// GetUserAgent returns the User-Agent header value
func (r *Request) GetUserAgent() string {
	return r.Headers.Get("User-Agent")
}

// SetBody sets the request body and updates Content-Length
func (r *Request) SetBody(body []byte) {
	r.Body = body
	if len(body) > 0 {
		r.Headers.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	} else {
		r.Headers.Del("Content-Length")
	}
}

// IsHTTPS checks if the request is for HTTPS
func (r *Request) IsHTTPS() bool {
	return strings.HasPrefix(strings.ToLower(r.URL), "https://")
}
