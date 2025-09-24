package request

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// RawRequest represents a parsed HTTP request with exact formatting preservation
type RawRequest struct {
	Method        string                     // HTTP method (GET, POST, etc.)
	URL           string                     // Request URL/path
	Version       string                     // HTTP version (HTTP/1.1, HTTP/2, etc.)
	Headers       *headers.OrderedHeadersRaw // Headers with exact formatting preserved
	Body          []byte                     // Request body
	Raw           []byte                     // Original raw request data
	RequestLine   string                     // Exact original request line
	HeaderSection []byte                     // Exact original header section
	BodySection   []byte                     // Exact original body section
}

// NewRawRequest creates a new RawRequest instance
func NewRawRequest() *RawRequest {
	return &RawRequest{
		Headers: headers.NewOrderedHeadersRaw(),
	}
}

// ParseRaw parses raw HTTP request data preserving exact formatting
func ParseRaw(data []byte) (*RawRequest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty request data")
	}

	req := NewRawRequest()
	req.Raw = make([]byte, len(data))
	copy(req.Raw, data)

	// Split into sections: request line, headers, body
	sections := bytes.SplitN(data, []byte("\n"), 2)
	if len(sections) < 1 {
		return nil, fmt.Errorf("no request line found")
	}

	// Parse request line (preserve exact format)
	req.RequestLine = strings.TrimRight(string(sections[0]), "\r")
	if err := req.parseRequestLineRaw(req.RequestLine); err != nil {
		return nil, err
	}

	if len(sections) > 1 {
		remaining := sections[1]

		// Find end of headers (double newline or end of data)
		headerEnd := bytes.Index(remaining, []byte("\n\n"))
		crlfEnd := bytes.Index(remaining, []byte("\r\n\r\n"))

		if crlfEnd != -1 && (headerEnd == -1 || crlfEnd < headerEnd) {
			headerEnd = crlfEnd
			req.HeaderSection = remaining[:headerEnd+2] // Include first \r\n
			if headerEnd+4 < len(remaining) {
				req.BodySection = remaining[headerEnd+4:]
			}
		} else if headerEnd != -1 {
			req.HeaderSection = remaining[:headerEnd]
			if headerEnd+2 < len(remaining) {
				req.BodySection = remaining[headerEnd+2:]
			}
		} else {
			// No body separator found, treat all as headers
			req.HeaderSection = remaining
		}

		// Parse headers preserving formatting
		if len(req.HeaderSection) > 0 {
			parsedHeaders, err := headers.ParseHeadersRaw(req.HeaderSection)
			if err != nil {
				// Continue with empty headers on parse error (fault tolerance)
				req.Headers = headers.NewOrderedHeadersRaw()
			} else {
				req.Headers = parsedHeaders
			}
		}

		// Set body
		req.Body = req.BodySection
	}

	return req, nil
}

// parseRequestLineRaw parses the HTTP request line with fault tolerance
func (r *RawRequest) parseRequestLineRaw(line string) error {
	parts := strings.Fields(line)

	if len(parts) < 2 {
		return fmt.Errorf("invalid request line format")
	}

	// Method
	r.Method = strings.ToUpper(parts[0])
	if r.Method == "" {
		return fmt.Errorf("empty HTTP method")
	}

	// URL/Path
	r.URL = parts[1]
	if r.URL == "" {
		return fmt.Errorf("empty URL/path")
	}

	// Version (optional, default to HTTP/1.1)
	if len(parts) >= 3 {
		r.Version = parts[2]
	} else {
		r.Version = "HTTP/1.1" // Default version for fault tolerance
	}

	// Validate version format
	if !strings.HasPrefix(strings.ToUpper(r.Version), "HTTP/") {
		// Keep the invalid version but mark as fault tolerance
		r.Version = "HTTP/1.1"
	}

	return nil
}

// BuildRaw reconstructs the HTTP request with exact formatting preservation
func (r *RawRequest) BuildRaw() []byte {
	var buf bytes.Buffer

	// Request line (exact format)
	buf.WriteString(r.RequestLine)
	buf.WriteString("\n")

	// Headers (exact formatting preserved)
	if r.Headers.Len() > 0 {
		headerBytes := r.Headers.BuildRaw()
		buf.Write(headerBytes)
	}

	// Empty line between headers and body
	buf.WriteString("\n")

	// Body (exact format)
	if len(r.BodySection) > 0 {
		buf.Write(r.BodySection)
	}

	return buf.Bytes()
}

// BuildRawString reconstructs the HTTP request as a string
func (r *RawRequest) BuildRawString() string {
	return string(r.BuildRaw())
}

// Clone creates a deep copy of the raw request
func (r *RawRequest) Clone() *RawRequest {
	clone := NewRawRequest()
	clone.Method = r.Method
	clone.URL = r.URL
	clone.Version = r.Version
	clone.RequestLine = r.RequestLine

	clone.Body = make([]byte, len(r.Body))
	copy(clone.Body, r.Body)

	clone.Raw = make([]byte, len(r.Raw))
	copy(clone.Raw, r.Raw)

	clone.HeaderSection = make([]byte, len(r.HeaderSection))
	copy(clone.HeaderSection, r.HeaderSection)

	clone.BodySection = make([]byte, len(r.BodySection))
	copy(clone.BodySection, r.BodySection)

	// Clone headers
	for _, header := range r.Headers.All() {
		clone.Headers.Set(header.Name, header.Value)
	}

	return clone
}

// ToStandard converts RawRequest to standard Request for compatibility
func (r *RawRequest) ToStandard() *Request {
	req := NewRequest()
	req.Method = r.Method
	req.URL = r.URL
	req.Version = r.Version
	req.Body = r.Body
	req.Raw = r.Raw

	// Convert headers
	for _, header := range r.Headers.AllStandard() {
		req.Headers.Set(header.Name, header.Value)
	}

	return req
}

// FromStandard creates RawRequest from standard Request (loses formatting)
func FromStandard(req *Request) *RawRequest {
	rawReq := NewRawRequest()
	rawReq.Method = req.Method
	rawReq.URL = req.URL
	rawReq.Version = req.Version
	rawReq.Body = req.Body
	rawReq.Raw = req.Raw

	// Build standard request line
	rawReq.RequestLine = fmt.Sprintf("%s %s %s", req.Method, req.URL, req.Version)

	// Convert headers (loses original formatting)
	for _, header := range req.Headers.All() {
		rawReq.Headers.Set(header.Name, header.Value)
	}

	rawReq.BodySection = req.Body

	return rawReq
}
