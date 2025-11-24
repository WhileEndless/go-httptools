// Package http2 provides HTTP/2 message representation for external library integration.
// This package provides a structured format (not binary frames) that can be used
// with HTTP/2 client/server libraries like golang.org/x/net/http2.
//
// HTTP/2 uses pseudo-headers (prefixed with ':') for request/response metadata:
//   - :method - HTTP method (GET, POST, etc.)
//   - :scheme - URI scheme (http, https)
//   - :authority - Host:port (equivalent to Host header in HTTP/1.1)
//   - :path - Request path with query string
//   - :status - Response status code (response only)
//
// Header order is preserved throughout all operations, even though HTTP/2 binary
// encoding doesn't require specific ordering.
package http2

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// HeaderField represents a single HTTP/2 header field
// Header order is preserved when iterating
type HeaderField struct {
	Name  string `json:"name"`
	Value string `json:"value"`

	// Sensitive marks the header as sensitive (won't be indexed in HPACK)
	Sensitive bool `json:"sensitive,omitempty"`
}

// HeaderList is an ordered list of headers that preserves insertion order
type HeaderList struct {
	fields []HeaderField
}

// NewHeaderList creates a new empty header list
func NewHeaderList() *HeaderList {
	return &HeaderList{
		fields: make([]HeaderField, 0),
	}
}

// Add appends a header field to the list
func (h *HeaderList) Add(name, value string) {
	h.fields = append(h.fields, HeaderField{Name: name, Value: value})
}

// AddSensitive appends a sensitive header field
func (h *HeaderList) AddSensitive(name, value string) {
	h.fields = append(h.fields, HeaderField{Name: name, Value: value, Sensitive: true})
}

// Set sets a header value, replacing any existing values
func (h *HeaderList) Set(name, value string) {
	nameLower := strings.ToLower(name)
	found := false

	for i, f := range h.fields {
		if strings.ToLower(f.Name) == nameLower {
			if !found {
				h.fields[i].Value = value
				found = true
			} else {
				// Remove duplicate
				h.fields = append(h.fields[:i], h.fields[i+1:]...)
			}
		}
	}

	if !found {
		h.Add(name, value)
	}
}

// Get returns the first value for the header name (case-insensitive)
func (h *HeaderList) Get(name string) string {
	nameLower := strings.ToLower(name)
	for _, f := range h.fields {
		if strings.ToLower(f.Name) == nameLower {
			return f.Value
		}
	}
	return ""
}

// GetAll returns all values for the header name (case-insensitive)
func (h *HeaderList) GetAll(name string) []string {
	nameLower := strings.ToLower(name)
	var values []string
	for _, f := range h.fields {
		if strings.ToLower(f.Name) == nameLower {
			values = append(values, f.Value)
		}
	}
	return values
}

// Del removes all headers with the given name (case-insensitive)
func (h *HeaderList) Del(name string) {
	nameLower := strings.ToLower(name)
	filtered := make([]HeaderField, 0, len(h.fields))
	for _, f := range h.fields {
		if strings.ToLower(f.Name) != nameLower {
			filtered = append(filtered, f)
		}
	}
	h.fields = filtered
}

// Has checks if a header exists (case-insensitive)
func (h *HeaderList) Has(name string) bool {
	nameLower := strings.ToLower(name)
	for _, f := range h.fields {
		if strings.ToLower(f.Name) == nameLower {
			return true
		}
	}
	return false
}

// Len returns the number of header fields
func (h *HeaderList) Len() int {
	return len(h.fields)
}

// All returns all header fields in order
func (h *HeaderList) All() []HeaderField {
	return h.fields
}

// Clone creates a deep copy of the header list
func (h *HeaderList) Clone() *HeaderList {
	clone := NewHeaderList()
	clone.fields = make([]HeaderField, len(h.fields))
	copy(clone.fields, h.fields)
	return clone
}

// InsertAt inserts a header at the specified position
func (h *HeaderList) InsertAt(index int, name, value string) {
	if index < 0 {
		index = 0
	}
	if index > len(h.fields) {
		index = len(h.fields)
	}

	h.fields = append(h.fields, HeaderField{})
	copy(h.fields[index+1:], h.fields[index:])
	h.fields[index] = HeaderField{Name: name, Value: value}
}

// InsertBefore inserts a header before the first occurrence of beforeName
func (h *HeaderList) InsertBefore(beforeName, name, value string) {
	beforeLower := strings.ToLower(beforeName)
	for i, f := range h.fields {
		if strings.ToLower(f.Name) == beforeLower {
			h.InsertAt(i, name, value)
			return
		}
	}
	// If not found, append
	h.Add(name, value)
}

// InsertAfter inserts a header after the first occurrence of afterName
func (h *HeaderList) InsertAfter(afterName, name, value string) {
	afterLower := strings.ToLower(afterName)
	for i, f := range h.fields {
		if strings.ToLower(f.Name) == afterLower {
			h.InsertAt(i+1, name, value)
			return
		}
	}
	// If not found, append
	h.Add(name, value)
}

// MoveToFront moves a header to the front of the list
func (h *HeaderList) MoveToFront(name string) {
	nameLower := strings.ToLower(name)
	for i, f := range h.fields {
		if strings.ToLower(f.Name) == nameLower {
			// Remove and prepend
			field := h.fields[i]
			h.fields = append(h.fields[:i], h.fields[i+1:]...)
			h.fields = append([]HeaderField{field}, h.fields...)
			return
		}
	}
}

// MoveToBack moves a header to the back of the list
func (h *HeaderList) MoveToBack(name string) {
	nameLower := strings.ToLower(name)
	for i, f := range h.fields {
		if strings.ToLower(f.Name) == nameLower {
			// Remove and append
			field := h.fields[i]
			h.fields = append(h.fields[:i], h.fields[i+1:]...)
			h.fields = append(h.fields, field)
			return
		}
	}
}

// MarshalJSON implements json.Marshaler for ordered JSON output
func (h *HeaderList) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.fields)
}

// UnmarshalJSON implements json.Unmarshaler
func (h *HeaderList) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &h.fields)
}

// ============================================================================
// HTTP/2 Request
// ============================================================================

// Request represents an HTTP/2 request
type Request struct {
	// Pseudo-headers (RFC 7540)
	Method    string `json:":method"`
	Scheme    string `json:":scheme"`
	Authority string `json:":authority"`
	Path      string `json:":path"`

	// Regular headers (order preserved)
	Headers *HeaderList `json:"headers"`

	// Request body
	Body    []byte `json:"body,omitempty"`
	RawBody []byte `json:"raw_body,omitempty"` // Original body (if compressed)

	// Stream ID (for multiplexing)
	StreamID uint32 `json:"stream_id,omitempty"`

	// Priority information
	Priority *Priority `json:"priority,omitempty"`

	// End stream flag (true if no body will follow)
	EndStream bool `json:"end_stream,omitempty"`
}

// Priority represents HTTP/2 stream priority
type Priority struct {
	StreamDependency uint32 `json:"stream_dependency,omitempty"`
	Weight           uint8  `json:"weight,omitempty"`
	Exclusive        bool   `json:"exclusive,omitempty"`
}

// NewRequest creates a new HTTP/2 request
func NewRequest() *Request {
	return &Request{
		Method:  "GET",
		Scheme:  "https",
		Path:    "/",
		Headers: NewHeaderList(),
	}
}

// Clone creates a deep copy of the request
func (r *Request) Clone() *Request {
	clone := NewRequest()
	clone.Method = r.Method
	clone.Scheme = r.Scheme
	clone.Authority = r.Authority
	clone.Path = r.Path
	clone.StreamID = r.StreamID
	clone.EndStream = r.EndStream

	clone.Headers = r.Headers.Clone()

	if len(r.Body) > 0 {
		clone.Body = make([]byte, len(r.Body))
		copy(clone.Body, r.Body)
	}

	if len(r.RawBody) > 0 {
		clone.RawBody = make([]byte, len(r.RawBody))
		copy(clone.RawBody, r.RawBody)
	}

	if r.Priority != nil {
		clone.Priority = &Priority{
			StreamDependency: r.Priority.StreamDependency,
			Weight:           r.Priority.Weight,
			Exclusive:        r.Priority.Exclusive,
		}
	}

	return clone
}

// GetAllHeaders returns all headers including pseudo-headers in proper order
// Pseudo-headers come first, followed by regular headers
func (r *Request) GetAllHeaders() []HeaderField {
	var all []HeaderField

	// Pseudo-headers first (in RFC 7540 recommended order)
	if r.Method != "" {
		all = append(all, HeaderField{Name: ":method", Value: r.Method})
	}
	if r.Scheme != "" {
		all = append(all, HeaderField{Name: ":scheme", Value: r.Scheme})
	}
	if r.Authority != "" {
		all = append(all, HeaderField{Name: ":authority", Value: r.Authority})
	}
	if r.Path != "" {
		all = append(all, HeaderField{Name: ":path", Value: r.Path})
	}

	// Regular headers
	all = append(all, r.Headers.All()...)

	return all
}

// SetHost sets the :authority pseudo-header and Host header
func (r *Request) SetHost(host string) {
	r.Authority = host
	r.Headers.Set("host", host)
}

// GetHost returns the host (from :authority or Host header)
func (r *Request) GetHost() string {
	if r.Authority != "" {
		return r.Authority
	}
	return r.Headers.Get("host")
}

// ToJSON returns JSON representation
func (r *Request) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// FromJSON parses JSON representation
func (r *Request) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}

// BuildHeaderBlock returns headers in wire format order
// This can be used with HPACK encoder
func (r *Request) BuildHeaderBlock() []HeaderField {
	return r.GetAllHeaders()
}

// ============================================================================
// HTTP/2 Response
// ============================================================================

// Response represents an HTTP/2 response
type Response struct {
	// Pseudo-header (RFC 7540)
	Status int `json:":status"`

	// Regular headers (order preserved)
	Headers *HeaderList `json:"headers"`

	// Response body
	Body       []byte `json:"body,omitempty"`
	RawBody    []byte `json:"raw_body,omitempty"`   // Original body (if compressed/chunked)
	Compressed bool   `json:"compressed,omitempty"` // Whether body was compressed

	// Stream ID (for multiplexing)
	StreamID uint32 `json:"stream_id,omitempty"`

	// End stream flag
	EndStream bool `json:"end_stream,omitempty"`
}

// NewResponse creates a new HTTP/2 response
func NewResponse() *Response {
	return &Response{
		Status:  200,
		Headers: NewHeaderList(),
	}
}

// Clone creates a deep copy of the response
func (r *Response) Clone() *Response {
	clone := NewResponse()
	clone.Status = r.Status
	clone.StreamID = r.StreamID
	clone.EndStream = r.EndStream
	clone.Compressed = r.Compressed

	clone.Headers = r.Headers.Clone()

	if len(r.Body) > 0 {
		clone.Body = make([]byte, len(r.Body))
		copy(clone.Body, r.Body)
	}

	if len(r.RawBody) > 0 {
		clone.RawBody = make([]byte, len(r.RawBody))
		copy(clone.RawBody, r.RawBody)
	}

	return clone
}

// GetAllHeaders returns all headers including pseudo-headers in proper order
func (r *Response) GetAllHeaders() []HeaderField {
	var all []HeaderField

	// Pseudo-header first
	all = append(all, HeaderField{Name: ":status", Value: strconv.Itoa(r.Status)})

	// Regular headers
	all = append(all, r.Headers.All()...)

	return all
}

// GetStatusText returns a human-readable status text
func (r *Response) GetStatusText() string {
	switch r.Status {
	case 100:
		return "Continue"
	case 101:
		return "Switching Protocols"
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 304:
		return "Not Modified"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	default:
		return fmt.Sprintf("Status %d", r.Status)
	}
}

// ToJSON returns JSON representation
func (r *Response) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// FromJSON parses JSON representation
func (r *Response) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}

// BuildHeaderBlock returns headers in wire format order
func (r *Response) BuildHeaderBlock() []HeaderField {
	return r.GetAllHeaders()
}
