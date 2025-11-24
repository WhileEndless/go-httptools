package request

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/chunked"
	"github.com/WhileEndless/go-httptools/pkg/cookies"
	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// Request represents a parsed HTTP request
type Request struct {
	Method  string                  // HTTP method (GET, POST, etc.)
	URL     string                  // Request URL/path (full URL with query string)
	Version string                  // HTTP version (HTTP/1.1, HTTP/2, etc.)
	Headers *headers.OrderedHeaders // Headers with preserved order
	Body    []byte                  // Request body
	Raw     []byte                  // Original raw request data

	// Line ending preservation
	LineSeparator string // Original line separator (\r\n or \n)

	// Transfer encoding
	TransferEncoding []string // Parsed from Transfer-Encoding header
	RawBody          []byte   // Original body (if chunked encoded)
	IsBodyChunked    bool     // Whether body is chunked encoded

	// Query parameters
	Path        string     // URL path without query string
	QueryParams url.Values // Parsed query parameters

	// Cookies
	Cookies []cookies.Cookie // Parsed from Cookie header

	// HTTP/2 specific
	PseudoHeaders map[string]string // :method, :path, :authority, :scheme
}

// NewRequest creates a new Request instance
func NewRequest() *Request {
	return &Request{
		Headers:          headers.NewOrderedHeaders(),
		LineSeparator:    "\r\n", // Default to CRLF
		TransferEncoding: []string{},
		QueryParams:      url.Values{},
		Cookies:          []cookies.Cookie{},
		PseudoHeaders:    make(map[string]string),
	}
}

// Clone creates a deep copy of the request
func (r *Request) Clone() *Request {
	clone := NewRequest()
	clone.Method = r.Method
	clone.URL = r.URL
	clone.Version = r.Version
	clone.Path = r.Path
	clone.IsBodyChunked = r.IsBodyChunked
	clone.LineSeparator = r.LineSeparator

	clone.Body = make([]byte, len(r.Body))
	copy(clone.Body, r.Body)

	clone.Raw = make([]byte, len(r.Raw))
	copy(clone.Raw, r.Raw)

	clone.RawBody = make([]byte, len(r.RawBody))
	copy(clone.RawBody, r.RawBody)

	// Clone headers
	for _, header := range r.Headers.All() {
		clone.Headers.Set(header.Name, header.Value)
	}

	// Clone transfer encoding
	clone.TransferEncoding = make([]string, len(r.TransferEncoding))
	copy(clone.TransferEncoding, r.TransferEncoding)

	// Clone query params
	clone.QueryParams = url.Values{}
	for key, values := range r.QueryParams {
		clone.QueryParams[key] = make([]string, len(values))
		copy(clone.QueryParams[key], values)
	}

	// Clone cookies
	clone.Cookies = make([]cookies.Cookie, len(r.Cookies))
	copy(clone.Cookies, r.Cookies)

	// Clone pseudo headers
	for key, value := range r.PseudoHeaders {
		clone.PseudoHeaders[key] = value
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

// ============================================================================
// Chunked Transfer Encoding
// ============================================================================

// DecodeChunkedBody decodes the body if it's chunked encoded
// Returns trailers found after final chunk
// Updates Body field with decoded data and sets IsBodyChunked to false
func (r *Request) DecodeChunkedBody() map[string]string {
	if !r.IsBodyChunked {
		// Body is not chunked, nothing to do
		return nil
	}

	// Decode using chunked package
	decodedBody, trailers := chunked.Decode(r.Body)

	// Store original chunked body in RawBody
	if len(r.RawBody) == 0 {
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
func (r *Request) EncodeChunkedBody(chunkSize int) {
	if r.IsBodyChunked {
		// Already chunked, nothing to do
		return
	}

	// Store original body
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
// Query Parameters
// ============================================================================

// ParseQueryParams extracts query parameters from URL
// Updates Path and QueryParams fields
func (r *Request) ParseQueryParams() {
	if r.URL == "" {
		return
	}

	// Find query string separator
	idx := strings.Index(r.URL, "?")
	if idx == -1 {
		// No query string
		r.Path = r.URL
		r.QueryParams = url.Values{}
		return
	}

	r.Path = r.URL[:idx]
	queryString := r.URL[idx+1:]

	// Parse query string
	params, err := url.ParseQuery(queryString)
	if err != nil {
		// Best effort: use empty values on error
		r.QueryParams = url.Values{}
		return
	}

	r.QueryParams = params
}

// GetQueryParam returns first value for query parameter key
func (r *Request) GetQueryParam(key string) string {
	return r.QueryParams.Get(key)
}

// GetQueryParams returns all values for query parameter key
func (r *Request) GetQueryParams(key string) []string {
	return r.QueryParams[key]
}

// SetQueryParam sets query parameter (replaces existing)
func (r *Request) SetQueryParam(key, value string) {
	r.QueryParams.Set(key, value)
}

// AddQueryParam adds query parameter (allows duplicates)
func (r *Request) AddQueryParam(key, value string) {
	r.QueryParams.Add(key, value)
}

// DeleteQueryParam removes query parameter
func (r *Request) DeleteQueryParam(key string) {
	r.QueryParams.Del(key)
}

// RebuildURL rebuilds URL from Path and QueryParams
// This must be called after modifying query parameters
func (r *Request) RebuildURL() {
	if r.Path == "" {
		r.Path = r.URL
	}

	if len(r.QueryParams) == 0 {
		r.URL = r.Path
		return
	}

	// Rebuild URL with query string
	r.URL = r.Path + "?" + r.QueryParams.Encode()
}

// ============================================================================
// HTTP/2 Pseudo-Headers
// ============================================================================

// GetPseudoHeader returns pseudo-header value (or empty string)
func (r *Request) GetPseudoHeader(name string) string {
	if !strings.HasPrefix(name, ":") {
		name = ":" + name
	}
	return r.PseudoHeaders[name]
}

// SetPseudoHeader sets pseudo-header (e.g., ":path", ":method")
func (r *Request) SetPseudoHeader(name, value string) {
	if !strings.HasPrefix(name, ":") {
		name = ":" + name
	}
	r.PseudoHeaders[name] = value
}

// BuildHTTP2 builds HTTP/2 format request
// Pseudo-headers come first, then regular headers
func (r *Request) BuildHTTP2() []byte {
	var result strings.Builder

	// Write pseudo-headers first (in specific order)
	pseudoOrder := []string{":method", ":scheme", ":authority", ":path"}
	for _, name := range pseudoOrder {
		if value, ok := r.PseudoHeaders[name]; ok {
			result.WriteString(fmt.Sprintf("%s: %s\r\n", name, value))
		}
	}

	// Write any other pseudo-headers
	for name, value := range r.PseudoHeaders {
		if name != ":method" && name != ":scheme" && name != ":authority" && name != ":path" {
			result.WriteString(fmt.Sprintf("%s: %s\r\n", name, value))
		}
	}

	// Write regular headers
	for _, header := range r.Headers.All() {
		result.WriteString(fmt.Sprintf("%s: %s\r\n", header.Name, header.Value))
	}

	// Blank line separating headers from body
	result.WriteString("\r\n")

	// Add body if present
	if len(r.Body) > 0 {
		result.Write(r.Body)
	}

	return []byte(result.String())
}

// ============================================================================
// Cookies
// ============================================================================

// ParseCookies extracts cookies from Cookie header
// Updates Cookies field
func (r *Request) ParseCookies() {
	cookieHeader := r.Headers.Get("Cookie")
	if cookieHeader == "" {
		r.Cookies = []cookies.Cookie{}
		return
	}

	r.Cookies = cookies.ParseCookies(cookieHeader)
}

// GetCookie returns cookie value by name
func (r *Request) GetCookie(name string) string {
	for _, cookie := range r.Cookies {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

// SetCookie sets cookie value (updates Cookies slice)
// If cookie with same name exists, updates it; otherwise adds new cookie
func (r *Request) SetCookie(name, value string) {
	// Try to find existing cookie
	for i := range r.Cookies {
		if r.Cookies[i].Name == name {
			r.Cookies[i].Value = value
			return
		}
	}

	// Cookie not found, add new one
	r.Cookies = append(r.Cookies, cookies.Cookie{
		Name:  name,
		Value: value,
	})
}

// DeleteCookie removes cookie by name
func (r *Request) DeleteCookie(name string) {
	filtered := make([]cookies.Cookie, 0, len(r.Cookies))
	for _, cookie := range r.Cookies {
		if cookie.Name != name {
			filtered = append(filtered, cookie)
		}
	}
	r.Cookies = filtered
}

// UpdateCookieHeader rebuilds Cookie header from Cookies slice
// This must be called after modifying cookies
func (r *Request) UpdateCookieHeader() {
	if len(r.Cookies) == 0 {
		r.Headers.Del("Cookie")
		return
	}

	cookieHeader := cookies.BuildCookieHeader(r.Cookies)
	r.Headers.Set("Cookie", cookieHeader)
}
