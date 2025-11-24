package headers

import (
	"bytes"
	"strings"
)

// ParseHeaders parses raw HTTP headers with fault tolerance
// Preserves order, original formatting, and line endings
func ParseHeaders(data []byte) (*OrderedHeaders, error) {
	headers := NewOrderedHeaders()

	// Process byte by byte to preserve exact line endings
	i := 0
	for i < len(data) {
		// Find the end of current line and determine line ending
		lineStart := i
		lineEnd := i

		// Scan until we find a line ending or reach end of data
		for lineEnd < len(data) && data[lineEnd] != '\n' && data[lineEnd] != '\r' {
			lineEnd++
		}

		// Determine line ending type
		lineEnding := ""
		nextLineStart := lineEnd

		if lineEnd < len(data) {
			if data[lineEnd] == '\r' {
				// Could be \r, \r\n, \r\r\n, etc.
				endingStart := lineEnd
				for nextLineStart < len(data) && data[nextLineStart] == '\r' {
					nextLineStart++
				}
				if nextLineStart < len(data) && data[nextLineStart] == '\n' {
					nextLineStart++
				}
				lineEnding = string(data[endingStart:nextLineStart])
			} else if data[lineEnd] == '\n' {
				// Just \n
				lineEnding = "\n"
				nextLineStart = lineEnd + 1
			}
		}

		// Get the line content (without line ending)
		lineContent := string(data[lineStart:lineEnd])

		// Skip empty lines (end of headers)
		if len(strings.TrimSpace(lineContent)) == 0 {
			break
		}

		// Original line is content without line ending
		originalLine := lineContent

		// Find colon separator
		colonPos := strings.Index(lineContent, ":")
		if colonPos == -1 {
			// Invalid header format, but store it anyway for fault tolerance
			headers.SetWithOriginal("X-Malformed-Header", lineContent, originalLine, lineEnding)
			i = nextLineStart
			continue
		}

		// Parse name and value (trimmed for programmatic access)
		name := strings.TrimSpace(lineContent[:colonPos])
		value := strings.TrimSpace(lineContent[colonPos+1:])

		// Handle empty header name (fault tolerance)
		if name == "" {
			name = "X-Empty-Header-Name"
		}

		// Store with original formatting preserved
		headers.SetWithOriginal(name, value, originalLine, lineEnding)

		i = nextLineStart
	}

	return headers, nil
}

// Build reconstructs headers preserving original formatting when available
func (h *OrderedHeaders) Build() []byte {
	var buf bytes.Buffer

	for _, header := range h.All() {
		if header.OriginalLine != "" {
			// Use original line format
			buf.WriteString(header.OriginalLine)
			if header.LineEnding != "" {
				buf.WriteString(header.LineEnding)
			} else {
				buf.WriteString("\r\n") // Default line ending
			}
		} else {
			// Programmatically added header - use standard format
			buf.WriteString(header.Name)
			buf.WriteString(": ")
			buf.WriteString(header.Value)
			buf.WriteString("\r\n")
		}
	}

	return buf.Bytes()
}

// BuildNormalized reconstructs headers in standard format (Name: Value\r\n)
// Use this when you need consistent formatting regardless of original input
func (h *OrderedHeaders) BuildNormalized() []byte {
	var buf bytes.Buffer

	for _, header := range h.All() {
		buf.WriteString(header.Name)
		buf.WriteString(": ")
		buf.WriteString(header.Value)
		buf.WriteString("\r\n")
	}

	return buf.Bytes()
}
