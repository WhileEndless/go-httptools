package request

import (
	"bytes"
	"fmt"
)

// Build reconstructs the HTTP request from parsed components
// Preserves original line endings when available
func (r *Request) Build() []byte {
	var buf bytes.Buffer

	// Use original line separator or default to CRLF
	lineSep := r.LineSeparator
	if lineSep == "" {
		lineSep = "\r\n"
	}

	// Request line
	buf.WriteString(r.Method)
	buf.WriteString(" ")
	buf.WriteString(r.URL)
	buf.WriteString(" ")
	buf.WriteString(r.Version)
	buf.WriteString(lineSep)

	// Headers (in preserved order with original formatting)
	headerBytes := r.Headers.Build()
	buf.Write(headerBytes)

	// Empty line between headers and body (use same separator as headers)
	// Check if last header has a specific line ending, otherwise use lineSep
	allHeaders := r.Headers.All()
	if len(allHeaders) > 0 && allHeaders[len(allHeaders)-1].LineEnding != "" {
		// Use the same line ending as the last header for the blank line
		buf.WriteString(allHeaders[len(allHeaders)-1].LineEnding)
	} else {
		buf.WriteString(lineSep)
	}

	// Body
	if len(r.Body) > 0 {
		buf.Write(r.Body)
	}

	return buf.Bytes()
}

// BuildString reconstructs the HTTP request as a string
func (r *Request) BuildString() string {
	return string(r.Build())
}

// UpdateContentLength updates the Content-Length header based on body size
func (r *Request) UpdateContentLength() {
	if len(r.Body) > 0 {
		r.Headers.Set("Content-Length", fmt.Sprintf("%d", len(r.Body)))
	} else {
		r.Headers.Del("Content-Length")
	}
}
