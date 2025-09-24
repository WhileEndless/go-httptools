package request

import (
	"bytes"
	"fmt"
)

// Build reconstructs the HTTP request from parsed components
func (r *Request) Build() []byte {
	var buf bytes.Buffer

	// Request line
	buf.WriteString(r.Method)
	buf.WriteString(" ")
	buf.WriteString(r.URL)
	buf.WriteString(" ")
	buf.WriteString(r.Version)
	buf.WriteString("\r\n")

	// Headers (in preserved order)
	headerBytes := r.Headers.Build()
	buf.Write(headerBytes)

	// Empty line between headers and body
	buf.WriteString("\r\n")

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
