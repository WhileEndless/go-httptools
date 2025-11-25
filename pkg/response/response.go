package response

import (
	"fmt"
	"io"
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

// ============================================================================
// Streaming Support (io.Writer)
// ============================================================================

// WriteTo implements io.WriterTo interface for streaming large responses
// Writes the complete HTTP response (status line + headers + body) to the writer
// Returns the number of bytes written and any error encountered
func (r *Response) WriteTo(w io.Writer) (int64, error) {
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

// WriteHeadersTo writes only the status line and headers to the writer
// This is useful for streaming large bodies separately
// Returns the number of bytes written and any error encountered
func (r *Response) WriteHeadersTo(w io.Writer) (int64, error) {
	var total int64

	// Build status line
	statusLine := fmt.Sprintf("%s %d %s%s", r.Version, r.StatusCode, r.StatusText, r.LineSeparator)
	n, err := w.Write([]byte(statusLine))
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
func (r *Response) WriteBodyTo(w io.Writer) (int64, error) {
	if len(r.Body) == 0 {
		return 0, nil
	}
	n, err := w.Write(r.Body)
	return int64(n), err
}

// CopyBodyFrom reads from the provided reader and writes to the writer
// This is useful for streaming large bodies without loading into memory
// The body is NOT stored in the Response struct
// Returns the number of bytes copied and any error encountered
func (r *Response) CopyBodyFrom(src io.Reader, dst io.Writer) (int64, error) {
	return io.Copy(dst, src)
}

// WriteToWithBody writes the response headers followed by body from an external reader
// This is useful for building large responses without loading body into memory
// The body is streamed directly from src to dst
// Returns the total number of bytes written and any error encountered
func (r *Response) WriteToWithBody(dst io.Writer, bodyReader io.Reader) (int64, error) {
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

// WriteToWithBodyChunked writes the response with chunked transfer encoding
// Body is read from bodyReader and written as chunked transfer encoding
// chunkSize specifies the size of each chunk (0 = default 8192)
func (r *Response) WriteToWithBodyChunked(dst io.Writer, bodyReader io.Reader, chunkSize int) (int64, error) {
	var total int64

	// Clone response and set chunked encoding
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

	// Use a countingWriter to track bytes
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

// StreamingBody provides streaming access to HTTP response body
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
// based on the response headers
//
// If isChunked is true, wraps with chunked decoder
// If contentEncoding is set, wraps with appropriate decompressor
// Order: raw -> chunked decode -> decompress (matches HTTP specification)
//
// The returned StreamingBody must be closed when done
func (r *Response) WrapBodyReader(bodyReader io.Reader) (*StreamingBody, error) {
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
