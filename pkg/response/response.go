package response

import (
	"fmt"
	"strconv"

	"github.com/WhileEndless/go-httptools/pkg/compression"
	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// Response represents a parsed HTTP response
type Response struct {
	Version    string                  // HTTP version (HTTP/1.1, HTTP/2, etc.)
	StatusCode int                     // HTTP status code (200, 404, etc.)
	StatusText string                  // Status text (OK, Not Found, etc.)
	Headers    *headers.OrderedHeaders // Headers with preserved order
	Body       []byte                  // Decompressed response body
	RawBody    []byte                  // Original compressed body (if any)
	Raw        []byte                  // Original raw response data
	Compressed bool                    // Whether original body was compressed
}

// NewResponse creates a new Response instance
func NewResponse() *Response {
	return &Response{
		Headers: headers.NewOrderedHeaders(),
	}
}

// Clone creates a deep copy of the response
func (r *Response) Clone() *Response {
	clone := NewResponse()
	clone.Version = r.Version
	clone.StatusCode = r.StatusCode
	clone.StatusText = r.StatusText
	clone.Compressed = r.Compressed

	clone.Body = make([]byte, len(r.Body))
	copy(clone.Body, r.Body)

	clone.RawBody = make([]byte, len(r.RawBody))
	copy(clone.RawBody, r.RawBody)

	clone.Raw = make([]byte, len(r.Raw))
	copy(clone.Raw, r.Raw)

	// Clone headers
	for _, header := range r.Headers.All() {
		clone.Headers.Set(header.Name, header.Value)
	}

	return clone
}

// GetContentLength returns the Content-Length header value as integer
func (r *Response) GetContentLength() int {
	lengthStr := r.Headers.Get("Content-Length")
	if lengthStr == "" {
		return 0
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return 0
	}

	return length
}

// GetContentType returns the Content-Type header value
func (r *Response) GetContentType() string {
	return r.Headers.Get("Content-Type")
}

// GetContentEncoding returns the Content-Encoding header value
func (r *Response) GetContentEncoding() string {
	return r.Headers.Get("Content-Encoding")
}

// GetServer returns the Server header value
func (r *Response) GetServer() string {
	return r.Headers.Get("Server")
}

// SetBody sets the response body and updates Content-Length
// If compress is true, compresses the body based on Content-Encoding header
func (r *Response) SetBody(body []byte, compress bool) error {
	r.Body = body

	if compress && r.GetContentEncoding() != "" {
		compressionType := compression.DetectCompression(r.GetContentEncoding())
		if compressionType != compression.CompressionNone {
			compressedBody, err := compression.Compress(body, compressionType)
			if err != nil {
				return err
			}
			r.RawBody = compressedBody
			r.Compressed = true
		} else {
			r.RawBody = body
			r.Compressed = false
		}
	} else {
		r.RawBody = body
		r.Compressed = false
	}

	// Update Content-Length based on raw body size
	if len(r.RawBody) > 0 {
		r.Headers.Set("Content-Length", fmt.Sprintf("%d", len(r.RawBody)))
	} else {
		r.Headers.Del("Content-Length")
	}

	return nil
}

// IsSuccessful returns true if the response has a 2xx status code
func (r *Response) IsSuccessful() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsRedirect returns true if the response has a 3xx status code
func (r *Response) IsRedirect() bool {
	return r.StatusCode >= 300 && r.StatusCode < 400
}

// IsClientError returns true if the response has a 4xx status code
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the response has a 5xx status code
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}

// GetRedirectLocation returns the Location header for redirects
func (r *Response) GetRedirectLocation() string {
	if r.IsRedirect() {
		return r.Headers.Get("Location")
	}
	return ""
}
