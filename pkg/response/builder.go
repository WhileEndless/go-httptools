package response

import (
	"bytes"
	"fmt"
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

// UpdateContentLength updates the Content-Length header based on body size
// Uses RawBody size to maintain accuracy with compressed content
func (r *Response) UpdateContentLength() {
	if len(r.RawBody) > 0 {
		r.Headers.Set("Content-Length", fmt.Sprintf("%d", len(r.RawBody)))
	} else {
		r.Headers.Del("Content-Length")
	}
}
