package headers

import (
	"bufio"
	"bytes"
	"strings"
)

// ParseHeaders parses raw HTTP headers with fault tolerance
// Preserves order and handles non-standard headers gracefully
func ParseHeaders(data []byte) (*OrderedHeaders, error) {
	headers := NewOrderedHeaders()
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines (end of headers)
		if len(strings.TrimSpace(line)) == 0 {
			break
		}

		// Find colon separator
		colonPos := strings.Index(line, ":")
		if colonPos == -1 {
			// Invalid header format, but store it anyway for fault tolerance
			headers.Set("X-Malformed-Header", line)
			continue
		}

		name := strings.TrimSpace(line[:colonPos])
		value := strings.TrimSpace(line[colonPos+1:])

		// Handle empty header name (fault tolerance)
		if name == "" {
			name = "X-Empty-Header-Name"
		}

		headers.Set(name, value)
	}

	return headers, scanner.Err()
}

// Build reconstructs headers in their original order
func (h *OrderedHeaders) Build() []byte {
	var buf bytes.Buffer

	for _, header := range h.All() {
		buf.WriteString(header.Name)
		buf.WriteString(": ")
		buf.WriteString(header.Value)
		buf.WriteString("\r\n")
	}

	return buf.Bytes()
}
