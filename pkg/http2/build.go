package http2

import (
	"bytes"
	"strconv"
)

// Build constructs a text representation of the HTTP/2 request
// This is for display/debugging purposes and maintains header order
// Output format is similar to HTTP/1.1 but with pseudo-headers visible
func (r *Request) Build() []byte {
	var buf bytes.Buffer

	// Write pseudo-headers first (in RFC 7540 order)
	if r.Method != "" {
		buf.WriteString(":method: ")
		buf.WriteString(r.Method)
		buf.WriteString("\r\n")
	}
	if r.Scheme != "" {
		buf.WriteString(":scheme: ")
		buf.WriteString(r.Scheme)
		buf.WriteString("\r\n")
	}
	if r.Authority != "" {
		buf.WriteString(":authority: ")
		buf.WriteString(r.Authority)
		buf.WriteString("\r\n")
	}
	if r.Path != "" {
		buf.WriteString(":path: ")
		buf.WriteString(r.Path)
		buf.WriteString("\r\n")
	}

	// Write regular headers in order
	for _, hdr := range r.Headers.All() {
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString("\r\n")
	}

	// End of headers
	buf.WriteString("\r\n")

	// Body
	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildCompact builds a compact single-line representation
// Format: METHOD path [headers count] [body size]
func (r *Request) BuildCompact() string {
	var buf bytes.Buffer
	buf.WriteString(r.Method)
	buf.WriteString(" ")
	buf.WriteString(r.Path)
	buf.WriteString(" [")
	buf.WriteString(strconv.Itoa(r.Headers.Len()))
	buf.WriteString(" headers]")

	if len(r.Body) > 0 {
		buf.WriteString(" [")
		buf.WriteString(strconv.Itoa(len(r.Body)))
		buf.WriteString(" bytes]")
	}

	return buf.String()
}

// BuildHTTP1Style builds an HTTP/1.1-like representation
// This converts pseudo-headers back to their HTTP/1.1 equivalents
func (r *Request) BuildHTTP1Style() []byte {
	var buf bytes.Buffer

	// Request line
	buf.WriteString(r.Method)
	buf.WriteString(" ")
	buf.WriteString(r.Path)
	buf.WriteString(" HTTP/2\r\n")

	// Host header from :authority
	if r.Authority != "" {
		buf.WriteString("Host: ")
		buf.WriteString(r.Authority)
		buf.WriteString("\r\n")
	}

	// Regular headers
	for _, hdr := range r.Headers.All() {
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString("\r\n")
	}

	buf.WriteString("\r\n")

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// Build constructs a text representation of the HTTP/2 response
func (r *Response) Build() []byte {
	var buf bytes.Buffer

	// Write status pseudo-header
	buf.WriteString(":status: ")
	buf.WriteString(strconv.Itoa(r.Status))
	buf.WriteString("\r\n")

	// Write regular headers in order
	for _, hdr := range r.Headers.All() {
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString("\r\n")
	}

	buf.WriteString("\r\n")

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildCompact builds a compact single-line representation
func (r *Response) BuildCompact() string {
	var buf bytes.Buffer
	buf.WriteString(strconv.Itoa(r.Status))
	buf.WriteString(" ")
	buf.WriteString(r.GetStatusText())
	buf.WriteString(" [")
	buf.WriteString(strconv.Itoa(r.Headers.Len()))
	buf.WriteString(" headers]")

	if len(r.Body) > 0 {
		buf.WriteString(" [")
		buf.WriteString(strconv.Itoa(len(r.Body)))
		buf.WriteString(" bytes]")
	}

	return buf.String()
}

// BuildHTTP1Style builds an HTTP/1.1-like representation
func (r *Response) BuildHTTP1Style() []byte {
	var buf bytes.Buffer

	// Status line
	buf.WriteString("HTTP/2 ")
	buf.WriteString(strconv.Itoa(r.Status))
	buf.WriteString(" ")
	buf.WriteString(r.GetStatusText())
	buf.WriteString("\r\n")

	// Headers
	for _, hdr := range r.Headers.All() {
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString("\r\n")
	}

	buf.WriteString("\r\n")

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildWithLineSeparator builds with custom line separator
func (r *Request) BuildWithLineSeparator(sep string) []byte {
	var buf bytes.Buffer

	if r.Method != "" {
		buf.WriteString(":method: ")
		buf.WriteString(r.Method)
		buf.WriteString(sep)
	}
	if r.Scheme != "" {
		buf.WriteString(":scheme: ")
		buf.WriteString(r.Scheme)
		buf.WriteString(sep)
	}
	if r.Authority != "" {
		buf.WriteString(":authority: ")
		buf.WriteString(r.Authority)
		buf.WriteString(sep)
	}
	if r.Path != "" {
		buf.WriteString(":path: ")
		buf.WriteString(r.Path)
		buf.WriteString(sep)
	}

	for _, hdr := range r.Headers.All() {
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString(sep)
	}

	buf.WriteString(sep)

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildWithLineSeparator builds response with custom line separator
func (r *Response) BuildWithLineSeparator(sep string) []byte {
	var buf bytes.Buffer

	buf.WriteString(":status: ")
	buf.WriteString(strconv.Itoa(r.Status))
	buf.WriteString(sep)

	for _, hdr := range r.Headers.All() {
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString(sep)
	}

	buf.WriteString(sep)

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// ============================================================================
// HTTP/1.1 Conversion (Direct Build)
// ============================================================================

// BuildAsHTTP1 builds as a complete HTTP/1.1 request byte slice
// This is different from BuildHTTP1Style which shows "HTTP/2" in version
func (r *Request) BuildAsHTTP1() []byte {
	var buf bytes.Buffer

	// Request line (HTTP/1.1)
	buf.WriteString(r.Method)
	buf.WriteString(" ")
	buf.WriteString(r.Path)
	buf.WriteString(" HTTP/1.1\r\n")

	// Host header from :authority (must come first for HTTP/1.1)
	if r.Authority != "" {
		buf.WriteString("Host: ")
		buf.WriteString(r.Authority)
		buf.WriteString("\r\n")
	}

	// Regular headers in order
	for _, hdr := range r.Headers.All() {
		// Skip if host already added from :authority
		if r.Authority != "" && hdr.Name == "host" {
			continue
		}
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString("\r\n")
	}

	// Content-Length if body present and not already set
	if len(r.Body) > 0 && !r.Headers.Has("content-length") {
		buf.WriteString("Content-Length: ")
		buf.WriteString(strconv.Itoa(len(r.Body)))
		buf.WriteString("\r\n")
	}

	buf.WriteString("\r\n")

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildAsHTTP1WithSeparator builds as HTTP/1.1 with custom line separator
func (r *Request) BuildAsHTTP1WithSeparator(sep string) []byte {
	var buf bytes.Buffer

	buf.WriteString(r.Method)
	buf.WriteString(" ")
	buf.WriteString(r.Path)
	buf.WriteString(" HTTP/1.1")
	buf.WriteString(sep)

	if r.Authority != "" {
		buf.WriteString("Host: ")
		buf.WriteString(r.Authority)
		buf.WriteString(sep)
	}

	for _, hdr := range r.Headers.All() {
		if r.Authority != "" && hdr.Name == "host" {
			continue
		}
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString(sep)
	}

	if len(r.Body) > 0 && !r.Headers.Has("content-length") {
		buf.WriteString("Content-Length: ")
		buf.WriteString(strconv.Itoa(len(r.Body)))
		buf.WriteString(sep)
	}

	buf.WriteString(sep)

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildAsHTTP1 builds as a complete HTTP/1.1 response byte slice
func (r *Response) BuildAsHTTP1() []byte {
	var buf bytes.Buffer

	// Status line
	buf.WriteString("HTTP/1.1 ")
	buf.WriteString(strconv.Itoa(r.Status))
	buf.WriteString(" ")
	buf.WriteString(r.GetStatusText())
	buf.WriteString("\r\n")

	// Headers in order
	for _, hdr := range r.Headers.All() {
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString("\r\n")
	}

	// Content-Length if body present and not already set
	if len(r.Body) > 0 && !r.Headers.Has("content-length") {
		buf.WriteString("Content-Length: ")
		buf.WriteString(strconv.Itoa(len(r.Body)))
		buf.WriteString("\r\n")
	}

	buf.WriteString("\r\n")

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildAsHTTP1WithSeparator builds as HTTP/1.1 with custom line separator
func (r *Response) BuildAsHTTP1WithSeparator(sep string) []byte {
	var buf bytes.Buffer

	buf.WriteString("HTTP/1.1 ")
	buf.WriteString(strconv.Itoa(r.Status))
	buf.WriteString(" ")
	buf.WriteString(r.GetStatusText())
	buf.WriteString(sep)

	for _, hdr := range r.Headers.All() {
		buf.WriteString(hdr.Name)
		buf.WriteString(": ")
		buf.WriteString(hdr.Value)
		buf.WriteString(sep)
	}

	if len(r.Body) > 0 && !r.Headers.Has("content-length") {
		buf.WriteString("Content-Length: ")
		buf.WriteString(strconv.Itoa(len(r.Body)))
		buf.WriteString(sep)
	}

	buf.WriteString(sep)

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}
