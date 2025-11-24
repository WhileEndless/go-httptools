package response

import (
	"bytes"
	"fmt"
	"strings"
)

// Build reconstructs the HTTP response from parsed components
// Preserves original line endings when available
// Uses RawBody (potentially compressed) for accurate reconstruction
func (r *Response) Build() []byte {
	var buf bytes.Buffer

	// Use original line separator or default to CRLF
	lineSep := r.LineSeparator
	if lineSep == "" {
		lineSep = "\r\n"
	}

	// Status line
	buf.WriteString(r.Version)
	buf.WriteString(" ")
	buf.WriteString(fmt.Sprintf("%d", r.StatusCode))
	buf.WriteString(" ")
	buf.WriteString(r.StatusText)
	buf.WriteString(lineSep)

	// Headers (in preserved order with original formatting)
	headerBytes := r.Headers.Build()
	buf.Write(headerBytes)

	// Empty line between headers and body (use same separator as headers)
	allHeaders := r.Headers.All()
	if len(allHeaders) > 0 && allHeaders[len(allHeaders)-1].LineEnding != "" {
		buf.WriteString(allHeaders[len(allHeaders)-1].LineEnding)
	} else {
		buf.WriteString(lineSep)
	}

	// Body (use RawBody to maintain compression if it was originally compressed)
	if len(r.RawBody) > 0 {
		buf.Write(r.RawBody)
	}

	return buf.Bytes()
}

// BuildString reconstructs the HTTP response as a string
func (r *Response) BuildString() string {
	return string(r.Build())
}

// BuildDecompressed builds the response with decompressed body
// Removes Content-Encoding header and updates Content-Length
func (r *Response) BuildDecompressed() []byte {
	var buf bytes.Buffer

	// Status line
	buf.WriteString(r.Version)
	buf.WriteString(" ")
	buf.WriteString(fmt.Sprintf("%d", r.StatusCode))
	buf.WriteString(" ")
	buf.WriteString(r.StatusText)
	buf.WriteString("\r\n")

	// Headers (modify for decompressed version)
	for _, header := range r.Headers.All() {
		// Skip Content-Encoding for decompressed version
		if strings.ToLower(header.Name) == "content-encoding" {
			continue
		}
		// Update Content-Length for decompressed body
		if strings.ToLower(header.Name) == "content-length" {
			buf.WriteString(header.Name)
			buf.WriteString(": ")
			buf.WriteString(fmt.Sprintf("%d", len(r.Body)))
			buf.WriteString("\r\n")
		} else {
			buf.WriteString(header.Name)
			buf.WriteString(": ")
			buf.WriteString(header.Value)
			buf.WriteString("\r\n")
		}
	}

	// Empty line between headers and body
	buf.WriteString("\r\n")

	// Decompressed body
	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// UpdateContentLength updates the Content-Length header based on body size
// Uses RawBody size to maintain accuracy with compressed content
func (r *Response) UpdateContentLength() {
	if len(r.RawBody) > 0 {
		r.Headers.Set("Content-Length", fmt.Sprintf("%d", len(r.RawBody)))
	} else {
		r.Headers.Del("Content-Length")
	}
}
