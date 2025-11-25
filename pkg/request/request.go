package request

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/chunked"
	"github.com/WhileEndless/go-httptools/pkg/compression"
	"github.com/WhileEndless/go-httptools/pkg/cookies"
	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// Request represents a parsed HTTP request
type Request struct {
	Method  string                  // HTTP method (GET, POST, etc.)
	URL     string                  // Request URL/path (full URL with query string)
	Version string                  // HTTP version (HTTP/1.1, HTTP/2, etc.)
	Headers *headers.OrderedHeaders // Headers with preserved order
	Body    []byte                  // Request body (decompressed if was compressed)
	RawBody []byte                  // Original body (if chunked/compressed)
	Raw     []byte                  // Original raw request data

	// Body state
	Compressed          bool                        // Whether original body was compressed
	DetectedCompression compression.CompressionType // Detected compression type (via header or magic bytes)
	IsBodyChunked       bool                        // Whether body is chunked encoded

	// Line ending preservation
	LineSeparator string // Original line separator (\r\n or \n)

	// Transfer encoding
	TransferEncoding []string // Parsed from Transfer-Encoding header

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
	clone.Compressed = r.Compressed
	clone.DetectedCompression = r.DetectedCompression
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

// GetContentLength returns the Content-Length header value (trimmed)
func (r *Request) GetContentLength() string {
	return strings.TrimSpace(r.Headers.Get("Content-Length"))
}

// GetContentType returns the Content-Type header value (trimmed)
func (r *Request) GetContentType() string {
	return strings.TrimSpace(r.Headers.Get("Content-Type"))
}

// GetHost returns the Host header value (trimmed)
func (r *Request) GetHost() string {
	return strings.TrimSpace(r.Headers.Get("Host"))
}

// GetUserAgent returns the User-Agent header value (trimmed)
func (r *Request) GetUserAgent() string {
	return strings.TrimSpace(r.Headers.Get("User-Agent"))
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

// ============================================================================
// Cookies
// ============================================================================

// ParseCookies extracts cookies from Cookie header
// Updates Cookies field
func (r *Request) ParseCookies() {
	cookieHeader := strings.TrimSpace(r.Headers.Get("Cookie"))
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

// ============================================================================
// Streaming Support (io.Writer)
// ============================================================================

// WriteTo implements io.WriterTo interface for streaming large requests
// Writes the complete HTTP request (request line + headers + body) to the writer
// Returns the number of bytes written and any error encountered
func (r *Request) WriteTo(w io.Writer) (int64, error) {
	var total int64

	// Write headers first
	n, err := r.WriteHeadersTo(w)
	total += n
	if err != nil {
		return total, err
	}

	// Write body
	if len(r.Body) > 0 {
		written, err := w.Write(r.Body)
		total += int64(written)
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// WriteHeadersTo writes only the request line and headers to the writer
// This is useful for streaming large bodies separately
// Returns the number of bytes written and any error encountered
func (r *Request) WriteHeadersTo(w io.Writer) (int64, error) {
	var total int64

	// Build request line
	requestLine := fmt.Sprintf("%s %s %s%s", r.Method, r.URL, r.Version, r.LineSeparator)
	n, err := w.Write([]byte(requestLine))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Write headers
	headerBytes := r.Headers.Build()
	n, err = w.Write(headerBytes)
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Write header-body separator
	n, err = w.Write([]byte(r.LineSeparator))
	total += int64(n)
	if err != nil {
		return total, err
	}

	return total, nil
}

// WriteBodyTo writes only the body to the writer
// This is useful when headers have already been written
// Returns the number of bytes written and any error encountered
func (r *Request) WriteBodyTo(w io.Writer) (int64, error) {
	if len(r.Body) == 0 {
		return 0, nil
	}
	n, err := w.Write(r.Body)
	return int64(n), err
}

// CopyBodyFrom reads from the provided reader and writes to the writer
// This is useful for streaming large bodies without loading into memory
// The body is NOT stored in the Request struct
// Returns the number of bytes copied and any error encountered
func (r *Request) CopyBodyFrom(src io.Reader, dst io.Writer) (int64, error) {
	return io.Copy(dst, src)
}

// WriteToWithBody writes the request headers followed by body from an external reader
// This is useful for building large requests without loading body into memory
// The body is streamed directly from src to dst
// Returns the total number of bytes written and any error encountered
func (r *Request) WriteToWithBody(dst io.Writer, bodyReader io.Reader) (int64, error) {
	var total int64

	// Write headers first
	n, err := r.WriteHeadersTo(dst)
	total += n
	if err != nil {
		return total, err
	}

	// Stream body from reader
	if bodyReader != nil {
		copied, err := io.Copy(dst, bodyReader)
		total += copied
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// WriteToWithBodyChunked writes the request with chunked transfer encoding
// Body is read from bodyReader and written as chunked transfer encoding
// chunkSize specifies the size of each chunk (0 = default 8192)
func (r *Request) WriteToWithBodyChunked(dst io.Writer, bodyReader io.Reader, chunkSize int) (int64, error) {
	var total int64

	// Clone request and set chunked encoding
	clone := r.Clone()
	clone.Headers.Set("Transfer-Encoding", "chunked")
	clone.Headers.Del("Content-Length")

	// Write headers
	n, err := clone.WriteHeadersTo(dst)
	total += n
	if err != nil {
		return total, err
	}

	// Write body with chunked encoding
	chunkedWriter := chunked.NewEncodeWriter(dst, chunkSize)

	// Use a buffer to copy
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		nr, readErr := bodyReader.Read(buf)
		if nr > 0 {
			nw, writeErr := chunkedWriter.Write(buf[:nr])
			total += int64(nw)
			if writeErr != nil {
				return total, writeErr
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return total, readErr
		}
	}

	// Close chunked writer to write final chunk
	if err := chunkedWriter.Close(); err != nil {
		return total, err
	}

	return total, nil
}

// ============================================================================
// Streaming Body Reader
// ============================================================================

// StreamingBody provides streaming access to HTTP request body
// Supports automatic decompression and chunked decoding
// Also provides body operations like Search
type StreamingBody struct {
	reader       io.Reader
	closeFunc    func() error
	isChunked    bool
	isCompressed bool
	compType     compression.CompressionType
	totalRead    int64
}

// WrapBodyReader wraps a body reader with automatic decompression and/or chunked decoding
// based on the request headers
//
// If isChunked is true, wraps with chunked decoder
// If contentEncoding is set, wraps with appropriate decompressor
// Order: raw -> chunked decode -> decompress (matches HTTP specification)
//
// The returned StreamingBody must be closed when done
func (r *Request) WrapBodyReader(bodyReader io.Reader) (*StreamingBody, error) {
	var reader io.Reader = bodyReader
	var closers []func() error

	// First: decode chunked if needed (chunked is outermost encoding)
	if r.IsBodyChunked {
		reader = chunked.NewDecodeReader(reader)
	}

	// Second: decompress if needed
	contentEncoding := r.GetContentEncoding()
	compType := compression.DetectCompression(contentEncoding)
	if compType != compression.CompressionNone {
		decompReader, err := compression.NewDecompressReader(reader, compType)
		if err != nil {
			// Close any previously opened closers
			for _, closer := range closers {
				closer()
			}
			return nil, err
		}
		reader = decompReader
		closers = append(closers, decompReader.Close)
	}

	closeFunc := func() error {
		var lastErr error
		for i := len(closers) - 1; i >= 0; i-- {
			if err := closers[i](); err != nil {
				lastErr = err
			}
		}
		return lastErr
	}

	return &StreamingBody{
		reader:       reader,
		closeFunc:    closeFunc,
		isChunked:    r.IsBodyChunked,
		isCompressed: compType != compression.CompressionNone,
		compType:     compType,
	}, nil
}

// Read implements io.Reader interface
func (s *StreamingBody) Read(p []byte) (int, error) {
	n, err := s.reader.Read(p)
	s.totalRead += int64(n)
	return n, err
}

// Close closes the streaming body and releases any resources
func (s *StreamingBody) Close() error {
	if s.closeFunc != nil {
		return s.closeFunc()
	}
	return nil
}

// TotalRead returns the total number of bytes read so far
func (s *StreamingBody) TotalRead() int64 {
	return s.totalRead
}

// IsChunked returns true if the body was chunked encoded
func (s *StreamingBody) IsChunked() bool {
	return s.isChunked
}

// IsCompressed returns true if the body was compressed
func (s *StreamingBody) IsCompressed() bool {
	return s.isCompressed
}

// CompressionType returns the compression type used
func (s *StreamingBody) CompressionType() compression.CompressionType {
	return s.compType
}

// WriteTo implements io.WriterTo interface
// Writes all remaining body data to the writer
func (s *StreamingBody) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, s.reader)
}

// ReadAll reads all remaining body data into memory
// Use with caution for large bodies
func (s *StreamingBody) ReadAll() ([]byte, error) {
	return io.ReadAll(s.reader)
}

// Search searches for a pattern in the streaming body
// Returns the offset of the first match, or -1 if not found
// WARNING: This reads through the body and cannot be undone
// The body reader will be at EOF after this call
func (s *StreamingBody) Search(pattern []byte) (int64, error) {
	if len(pattern) == 0 {
		return -1, nil
	}

	// Use a sliding window approach for memory efficiency
	bufSize := 64 * 1024 // 64KB buffer
	if len(pattern) > bufSize/2 {
		bufSize = len(pattern) * 4
	}

	buf := make([]byte, bufSize)
	overlap := len(pattern) - 1
	offset := int64(0)
	buffered := 0

	for {
		// Read more data
		n, err := s.reader.Read(buf[buffered:])
		if n > 0 {
			buffered += n

			// Search in current buffer
			idx := searchBytes(buf[:buffered], pattern)
			if idx >= 0 {
				return offset + int64(idx), nil
			}

			// Keep overlap for next iteration
			if buffered > overlap {
				keep := overlap
				copy(buf, buf[buffered-keep:buffered])
				offset += int64(buffered - keep)
				buffered = keep
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return -1, err
		}
	}

	return -1, nil
}

// SearchString searches for a string pattern in the streaming body
func (s *StreamingBody) SearchString(pattern string) (int64, error) {
	return s.Search([]byte(pattern))
}

// Contains checks if the pattern exists in the streaming body
// WARNING: This reads through the body and cannot be undone
func (s *StreamingBody) Contains(pattern []byte) (bool, error) {
	offset, err := s.Search(pattern)
	return offset >= 0, err
}

// ContainsString checks if the string exists in the streaming body
func (s *StreamingBody) ContainsString(pattern string) (bool, error) {
	return s.Contains([]byte(pattern))
}

// CopyTo copies all remaining body data to the writer
// This is an alias for WriteTo for clearer semantics
func (s *StreamingBody) CopyTo(w io.Writer) (int64, error) {
	return s.WriteTo(w)
}

// searchBytes searches for pattern in data using a simple algorithm
// Returns the index of the first match, or -1 if not found
func searchBytes(data, pattern []byte) int {
	if len(pattern) == 0 || len(data) < len(pattern) {
		return -1
	}

	for i := 0; i <= len(data)-len(pattern); i++ {
		found := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}
