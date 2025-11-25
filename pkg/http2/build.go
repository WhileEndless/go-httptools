package http2

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// ============================================================================
// HTTP/2 Request Build Methods
// ============================================================================

// Build constructs an HTTP/2 request in raw HTTP format
// Output: GET /path HTTP/2\r\nHost: example.com\r\n\r\nbody
// This is the standard format for HTTP/2 requests in text representation
func (r *Request) Build() []byte {
	var buf bytes.Buffer

	// Request line: METHOD PATH HTTP/2
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

	// Regular headers in order
	for _, hdr := range r.Headers.All() {
		// Skip host if already added from :authority
		if r.Authority != "" && hdr.Name == "host" {
			continue
		}
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

// BuildString returns the request as a string
func (r *Request) BuildString() string {
	return string(r.Build())
}

// BuildWithLineSeparator builds with custom line separator
// Output: GET /path HTTP/2{sep}Host: example.com{sep}{sep}body
func (r *Request) BuildWithLineSeparator(sep string) []byte {
	var buf bytes.Buffer

	buf.WriteString(r.Method)
	buf.WriteString(" ")
	buf.WriteString(r.Path)
	buf.WriteString(" HTTP/2")
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

	buf.WriteString(sep)

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildPseudoHeaders constructs request in pseudo-header format
// Output: :method: GET\r\n:scheme: https\r\n:authority: example.com\r\n:path: /\r\n
// This format is used for HPACK encoding and debugging HTTP/2 internals
func (r *Request) BuildPseudoHeaders() []byte {
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

// BuildPseudoHeadersWithLineSeparator builds pseudo-header format with custom separator
func (r *Request) BuildPseudoHeadersWithLineSeparator(sep string) []byte {
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

// BuildAsHTTP1 builds as a complete HTTP/1.1 request byte slice
// Output: GET /path HTTP/1.1\r\nHost: example.com\r\n\r\nbody
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

// ============================================================================
// HTTP/2 Request Streaming Methods
// ============================================================================

// WriteTo writes the complete HTTP/2 request to the writer
// Implements io.WriterTo interface
func (r *Request) WriteTo(w io.Writer) (int64, error) {
	var total int64

	// Write headers
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
// Returns number of bytes written
func (r *Request) WriteHeadersTo(w io.Writer) (int64, error) {
	var total int64

	// Request line
	requestLine := fmt.Sprintf("%s %s HTTP/2\r\n", r.Method, r.Path)
	n, err := w.Write([]byte(requestLine))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Host header from :authority
	if r.Authority != "" {
		hostHeader := fmt.Sprintf("Host: %s\r\n", r.Authority)
		n, err = w.Write([]byte(hostHeader))
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// Regular headers
	for _, hdr := range r.Headers.All() {
		if r.Authority != "" && hdr.Name == "host" {
			continue
		}
		header := fmt.Sprintf("%s: %s\r\n", hdr.Name, hdr.Value)
		n, err = w.Write([]byte(header))
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// End of headers
	n, err = w.Write([]byte("\r\n"))
	total += int64(n)
	if err != nil {
		return total, err
	}

	return total, nil
}

// WriteToWithBody writes the request headers and streams body from an io.Reader
// This is useful for large bodies that shouldn't be loaded into memory
func (r *Request) WriteToWithBody(w io.Writer, bodyReader io.Reader) (int64, error) {
	var total int64

	// Write headers
	n, err := r.WriteHeadersTo(w)
	total += n
	if err != nil {
		return total, err
	}

	// Stream body from reader
	if bodyReader != nil {
		copied, err := io.Copy(w, bodyReader)
		total += copied
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// WriteAsHTTP1To writes the request as HTTP/1.1 to the writer
func (r *Request) WriteAsHTTP1To(w io.Writer) (int64, error) {
	data := r.BuildAsHTTP1()
	n, err := w.Write(data)
	return int64(n), err
}

// WriteAsHTTP1WithBodyTo writes the request as HTTP/1.1 with streaming body
func (r *Request) WriteAsHTTP1WithBodyTo(w io.Writer, bodyReader io.Reader, contentLength int64) (int64, error) {
	var total int64

	// Request line
	requestLine := fmt.Sprintf("%s %s HTTP/1.1\r\n", r.Method, r.Path)
	n, err := w.Write([]byte(requestLine))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Host header
	if r.Authority != "" {
		hostHeader := fmt.Sprintf("Host: %s\r\n", r.Authority)
		n, err = w.Write([]byte(hostHeader))
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// Regular headers
	for _, hdr := range r.Headers.All() {
		if r.Authority != "" && hdr.Name == "host" {
			continue
		}
		header := fmt.Sprintf("%s: %s\r\n", hdr.Name, hdr.Value)
		n, err = w.Write([]byte(header))
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// Content-Length for streaming body
	if contentLength > 0 && !r.Headers.Has("content-length") {
		clHeader := fmt.Sprintf("Content-Length: %d\r\n", contentLength)
		n, err = w.Write([]byte(clHeader))
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// End of headers
	n, err = w.Write([]byte("\r\n"))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Stream body
	if bodyReader != nil {
		copied, err := io.Copy(w, bodyReader)
		total += copied
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// ============================================================================
// HTTP/2 Response Build Methods
// ============================================================================

// Build constructs an HTTP/2 response in raw HTTP format
// Output: HTTP/2 200 OK\r\nContent-Type: text/html\r\n\r\nbody
func (r *Response) Build() []byte {
	var buf bytes.Buffer

	// Status line: HTTP/2 STATUS STATUS_TEXT
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

// BuildString returns the response as a string
func (r *Response) BuildString() string {
	return string(r.Build())
}

// BuildWithLineSeparator builds with custom line separator
func (r *Response) BuildWithLineSeparator(sep string) []byte {
	var buf bytes.Buffer

	buf.WriteString("HTTP/2 ")
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

	buf.WriteString(sep)

	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildPseudoHeaders constructs response in pseudo-header format
// Output: :status: 200\r\ncontent-type: text/html\r\n\r\nbody
// This format is used for HPACK encoding and debugging HTTP/2 internals
func (r *Response) BuildPseudoHeaders() []byte {
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

// BuildPseudoHeadersWithLineSeparator builds pseudo-header format with custom separator
func (r *Response) BuildPseudoHeadersWithLineSeparator(sep string) []byte {
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

// BuildAsHTTP1 builds as a complete HTTP/1.1 response byte slice
// Output: HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\nbody
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

// ============================================================================
// HTTP/2 Response Streaming Methods
// ============================================================================

// WriteTo writes the complete HTTP/2 response to the writer
// Implements io.WriterTo interface
func (r *Response) WriteTo(w io.Writer) (int64, error) {
	var total int64

	// Write headers
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
func (r *Response) WriteHeadersTo(w io.Writer) (int64, error) {
	var total int64

	// Status line
	statusLine := fmt.Sprintf("HTTP/2 %d %s\r\n", r.Status, r.GetStatusText())
	n, err := w.Write([]byte(statusLine))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Headers
	for _, hdr := range r.Headers.All() {
		header := fmt.Sprintf("%s: %s\r\n", hdr.Name, hdr.Value)
		n, err = w.Write([]byte(header))
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// End of headers
	n, err = w.Write([]byte("\r\n"))
	total += int64(n)
	if err != nil {
		return total, err
	}

	return total, nil
}

// WriteToWithBody writes the response headers and streams body from an io.Reader
func (r *Response) WriteToWithBody(w io.Writer, bodyReader io.Reader) (int64, error) {
	var total int64

	// Write headers
	n, err := r.WriteHeadersTo(w)
	total += n
	if err != nil {
		return total, err
	}

	// Stream body from reader
	if bodyReader != nil {
		copied, err := io.Copy(w, bodyReader)
		total += copied
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// WriteAsHTTP1To writes the response as HTTP/1.1 to the writer
func (r *Response) WriteAsHTTP1To(w io.Writer) (int64, error) {
	data := r.BuildAsHTTP1()
	n, err := w.Write(data)
	return int64(n), err
}

// ============================================================================
// Deprecated: Old method names kept for backward compatibility
// ============================================================================

// BuildHTTP1Style is deprecated: use Build() instead
// This method now returns the same output as Build()
func (r *Request) BuildHTTP1Style() []byte {
	return r.Build()
}

// BuildHTTP1Style is deprecated: use Build() instead
// This method now returns the same output as Build()
func (r *Response) BuildHTTP1Style() []byte {
	return r.Build()
}
