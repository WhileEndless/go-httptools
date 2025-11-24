package response

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/chunked"
	"github.com/WhileEndless/go-httptools/pkg/compression"
	"github.com/WhileEndless/go-httptools/pkg/cookies"
	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// Response represents a parsed HTTP response
type Response struct {
	Version    string                  // HTTP version (HTTP/1.1, HTTP/2, etc.)
	StatusCode int                     // HTTP status code (200, 404, etc.)
	StatusText string                  // Status text (OK, Not Found, etc.)
	Headers    *headers.OrderedHeaders // Headers with preserved order
	Body       []byte                  // Decompressed response body
	RawBody    []byte                  // Original compressed/chunked body (if any)
	Raw        []byte                  // Original raw response data
	Compressed bool                    // Whether original body was compressed

	// Compression detection
	DetectedCompression compression.CompressionType // Detected compression type (via header or magic bytes)

	// Line ending preservation
	LineSeparator string // Original line separator (\r\n or \n)

	// Transfer encoding
	TransferEncoding []string // Parsed from Transfer-Encoding header
	IsBodyChunked    bool     // Whether body is chunked encoded

	// Set-Cookie headers
	SetCookies []cookies.ResponseCookie // Parsed from Set-Cookie headers
}

// NewResponse creates a new Response instance
func NewResponse() *Response {
	return &Response{
		Headers:          headers.NewOrderedHeaders(),
		LineSeparator:    "\r\n", // Default to CRLF
		TransferEncoding: []string{},
		SetCookies:       []cookies.ResponseCookie{},
	}
}

// Clone creates a deep copy of the response
func (r *Response) Clone() *Response {
	clone := NewResponse()
	clone.Version = r.Version
	clone.StatusCode = r.StatusCode
	clone.StatusText = r.StatusText
	clone.Compressed = r.Compressed
	clone.DetectedCompression = r.DetectedCompression
	clone.IsBodyChunked = r.IsBodyChunked
	clone.LineSeparator = r.LineSeparator

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

	// Clone transfer encoding
	clone.TransferEncoding = make([]string, len(r.TransferEncoding))
	copy(clone.TransferEncoding, r.TransferEncoding)

	// Clone set-cookies
	clone.SetCookies = make([]cookies.ResponseCookie, len(r.SetCookies))
	copy(clone.SetCookies, r.SetCookies)

	return clone
}

// GetContentLength returns the Content-Length header value as integer
func (r *Response) GetContentLength() int {
	lengthStr := strings.TrimSpace(r.Headers.Get("Content-Length"))
	if lengthStr == "" {
		return 0
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return 0
	}

	return length
}

// GetContentType returns the Content-Type header value (trimmed)
func (r *Response) GetContentType() string {
	return strings.TrimSpace(r.Headers.Get("Content-Type"))
}

// GetContentEncoding returns the Content-Encoding header value (trimmed)
func (r *Response) GetContentEncoding() string {
	return strings.TrimSpace(r.Headers.Get("Content-Encoding"))
}

// GetServer returns the Server header value (trimmed)
func (r *Response) GetServer() string {
	return strings.TrimSpace(r.Headers.Get("Server"))
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

// GetRedirectLocation returns the Location header for redirects (trimmed)
func (r *Response) GetRedirectLocation() string {
	if r.IsRedirect() {
		return strings.TrimSpace(r.Headers.Get("Location"))
	}
	return ""
}

// ============================================================================
// Chunked Transfer Encoding
// ============================================================================

// DecodeChunkedBody decodes the body if it's chunked encoded
// Returns trailers found after final chunk
// Updates Body field with decoded data and sets IsBodyChunked to false
func (r *Response) DecodeChunkedBody() map[string]string {
	if !r.IsBodyChunked {
		// Body is not chunked, nothing to do
		return nil
	}

	// Decode using chunked package
	decodedBody, trailers := chunked.Decode(r.Body)

	// Store original chunked body in RawBody if not already stored
	if len(r.RawBody) == 0 || r.IsBodyChunked {
		r.RawBody = make([]byte, len(r.Body))
		copy(r.RawBody, r.Body)
	}

	// Update body with decoded version
	r.Body = decodedBody
	r.IsBodyChunked = false

	// Update Content-Length header (remove it, as chunked doesn't use it)
	r.Headers.Del("Content-Length")

	return trailers
}

// EncodeChunkedBody encodes the body with chunked transfer encoding
// chunkSize specifies the size of each chunk (0 = default 8192)
func (r *Response) EncodeChunkedBody(chunkSize int) {
	if r.IsBodyChunked {
		// Already chunked, nothing to do
		return
	}

	// Store original body if not already stored
	if len(r.RawBody) == 0 {
		r.RawBody = make([]byte, len(r.Body))
		copy(r.RawBody, r.Body)
	}

	// Encode body
	r.Body = chunked.Encode(r.Body, chunkSize)
	r.IsBodyChunked = true

	// Update headers
	r.Headers.Set("Transfer-Encoding", "chunked")
	r.Headers.Del("Content-Length")
}

// ============================================================================
// Set-Cookie Support
// ============================================================================

// ParseSetCookies extracts Set-Cookie headers
// Updates SetCookies field
func (r *Response) ParseSetCookies() {
	// Get all Set-Cookie headers (there can be multiple)
	allHeaders := r.Headers.All()
	r.SetCookies = []cookies.ResponseCookie{}

	for _, header := range allHeaders {
		if header.Name == "Set-Cookie" {
			cookie := cookies.ParseSetCookie(header.Value)
			r.SetCookies = append(r.SetCookies, cookie)
		}
	}
}

// GetSetCookie returns Set-Cookie by name
func (r *Response) GetSetCookie(name string) *cookies.ResponseCookie {
	for i := range r.SetCookies {
		if r.SetCookies[i].Name == name {
			return &r.SetCookies[i]
		}
	}
	return nil
}

// AddSetCookie adds Set-Cookie header
func (r *Response) AddSetCookie(cookie cookies.ResponseCookie) {
	r.SetCookies = append(r.SetCookies, cookie)
}

// DeleteSetCookie removes Set-Cookie by name
func (r *Response) DeleteSetCookie(name string) {
	filtered := make([]cookies.ResponseCookie, 0, len(r.SetCookies))
	for _, cookie := range r.SetCookies {
		if cookie.Name != name {
			filtered = append(filtered, cookie)
		}
	}
	r.SetCookies = filtered
}

// UpdateSetCookieHeaders rebuilds Set-Cookie headers from SetCookies slice
// This must be called after modifying Set-Cookie values
func (r *Response) UpdateSetCookieHeaders() {
	// Remove all existing Set-Cookie headers
	r.Headers.DelAll("Set-Cookie")

	// Add new Set-Cookie headers
	for _, cookie := range r.SetCookies {
		setCookieValue := cookie.Build()
		r.Headers.Add("Set-Cookie", setCookieValue)
	}
}
