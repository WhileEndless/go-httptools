package http2

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// Parse parses raw HTTP/2 request data
// Supports two formats:
//
// Format 1: HTTP/1.1-style with HTTP/2 version
//
//	GET /path HTTP/2
//	Host: example.com
//	User-Agent: test
//
//	body
//
// Format 2: Pseudo-header format
//
//	:method: GET
//	:scheme: https
//	:authority: example.com
//	:path: /path
//	user-agent: test
//
//	body
//
// Both formats preserve header order and support body parsing.
func Parse(data []byte) (*Request, error) {
	if len(data) == 0 {
		return nil, &ParseError{Message: "empty request data"}
	}

	// Detect line separator
	lineSep := detectLineSeparator(data)

	// Find header end
	headerEnd, sepLen := findHeaderEnd(data, lineSep)
	var headerSection, bodySection []byte

	if headerEnd >= 0 {
		headerSection = data[:headerEnd]
		bodyStart := headerEnd + sepLen
		if bodyStart < len(data) {
			bodySection = data[bodyStart:]
		}
	} else {
		headerSection = data
	}

	// Split into lines
	lines := splitLines(headerSection, lineSep)
	if len(lines) == 0 {
		return nil, &ParseError{Message: "no header lines found"}
	}

	// Detect format by checking first line
	firstLine := strings.TrimSpace(lines[0])
	if strings.HasPrefix(firstLine, ":") {
		// Pseudo-header format
		return parsePseudoHeaderFormat(lines, bodySection)
	}

	// HTTP/1.1-style format
	return parseHTTPStyleFormat(lines, bodySection)
}

// ParseReader parses an HTTP/2 request from an io.Reader
func ParseReader(r io.Reader) (*Request, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &ParseError{Message: "failed to read from reader: " + err.Error()}
	}
	return Parse(data)
}

// ParseHeadersFromReader parses only HTTP/2 request headers from an io.Reader
// Returns the parsed Request (without body) and an io.Reader for the remaining body data
// Useful for streaming large bodies without loading into memory
func ParseHeadersFromReader(r io.Reader) (*Request, io.Reader, error) {
	br := bufio.NewReader(r)

	// Peek first line to detect format
	firstLine, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, nil, &ParseError{Message: "failed to read first line: " + err.Error()}
	}

	lineSep := "\n"
	if strings.HasSuffix(firstLine, "\r\n") {
		lineSep = "\r\n"
	}
	firstLine = strings.TrimRight(firstLine, "\r\n")

	var lines []string
	lines = append(lines, firstLine)

	// Read remaining header lines
	for {
		line, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, nil, &ParseError{Message: "failed to read header line: " + err.Error()}
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break // End of headers
		}

		lines = append(lines, trimmed)

		if err == io.EOF {
			break
		}
	}

	// Parse based on format
	var req *Request
	if strings.HasPrefix(strings.TrimSpace(lines[0]), ":") {
		req, err = parsePseudoHeaderFormat(lines, nil)
	} else {
		req, err = parseHTTPStyleFormat(lines, nil)
	}

	if err != nil {
		return nil, nil, err
	}

	// Store line separator info (useful for building)
	_ = lineSep // Could store in request if needed

	return req, br, nil
}

// parseHTTPStyleFormat parses HTTP/1.1-style format with HTTP/2 version
// Format: METHOD URL HTTP/2
func parseHTTPStyleFormat(lines []string, body []byte) (*Request, error) {
	req := NewRequest()

	if len(lines) == 0 {
		return nil, &ParseError{Message: "no request line"}
	}

	// Parse request line
	requestLine := strings.TrimSpace(lines[0])
	parts := strings.Fields(requestLine)

	if len(parts) < 2 {
		return nil, &ParseError{Message: "invalid request line format"}
	}

	req.Method = strings.ToUpper(parts[0])
	req.Path = parts[1]

	// Version check (optional, default to HTTP/2)
	if len(parts) >= 3 {
		version := strings.ToUpper(parts[2])
		// Accept HTTP/2, HTTP/2.0, or h2
		if !strings.Contains(version, "2") && version != "H2" {
			return nil, &ParseError{Message: "not an HTTP/2 request: " + parts[2]}
		}
	}

	// Parse headers
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx <= 0 {
			continue // Skip malformed headers
		}

		name := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		// Handle special headers
		nameLower := strings.ToLower(name)
		switch nameLower {
		case "host":
			req.Authority = value
		case ":method":
			req.Method = value
		case ":scheme":
			req.Scheme = value
		case ":authority":
			req.Authority = value
		case ":path":
			req.Path = value
		default:
			// Skip connection-specific headers (not used in HTTP/2)
			if nameLower == "connection" ||
				nameLower == "keep-alive" ||
				nameLower == "proxy-connection" ||
				nameLower == "transfer-encoding" ||
				nameLower == "upgrade" {
				continue
			}
			req.Headers.Add(name, value)
		}
	}

	// Set default scheme if not set
	if req.Scheme == "" {
		req.Scheme = "https"
	}

	// Set body
	if len(body) > 0 {
		req.Body = make([]byte, len(body))
		copy(req.Body, body)
		req.EndStream = false
	} else {
		req.EndStream = true
	}

	return req, nil
}

// parsePseudoHeaderFormat parses pseudo-header format
// Format: :method: GET, :path: /, etc.
func parsePseudoHeaderFormat(lines []string, body []byte) (*Request, error) {
	req := &Request{
		Headers: NewHeaderList(),
	}
	methodFound := false
	pathFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Find colon (skip first char if it's : for pseudo-headers)
		var colonIdx int
		if strings.HasPrefix(line, ":") {
			// Pseudo-header: find second colon
			colonIdx = strings.Index(line[1:], ":")
			if colonIdx >= 0 {
				colonIdx++ // Adjust for skipped first char
			}
		} else {
			colonIdx = strings.Index(line, ":")
		}

		if colonIdx <= 0 {
			continue // Skip malformed lines
		}

		name := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		// Handle pseudo-headers
		switch strings.ToLower(name) {
		case ":method":
			req.Method = value
			methodFound = true
		case ":scheme":
			req.Scheme = value
		case ":authority":
			req.Authority = value
		case ":path":
			req.Path = value
			pathFound = true
		case "host":
			// Also accept Host header, set as authority if not already set
			if req.Authority == "" {
				req.Authority = value
			}
		default:
			// Skip connection-specific headers
			nameLower := strings.ToLower(name)
			if nameLower == "connection" ||
				nameLower == "keep-alive" ||
				nameLower == "proxy-connection" ||
				nameLower == "transfer-encoding" ||
				nameLower == "upgrade" {
				continue
			}
			req.Headers.Add(name, value)
		}
	}

	// Validate required pseudo-headers
	if !methodFound {
		return nil, &ParseError{Message: "missing :method pseudo-header"}
	}
	if !pathFound {
		return nil, &ParseError{Message: "missing :path pseudo-header"}
	}

	// Set default scheme if not set
	if req.Scheme == "" {
		req.Scheme = "https"
	}

	// Set body
	if len(body) > 0 {
		req.Body = make([]byte, len(body))
		copy(req.Body, body)
		req.EndStream = false
	} else {
		req.EndStream = true
	}

	return req, nil
}

// ParseResponse parses raw HTTP/2 response data
// Supports two formats:
//
// Format 1: HTTP/1.1-style with HTTP/2 version
//
//	HTTP/2 200 OK
//	Content-Type: text/html
//
//	body
//
// Format 2: Pseudo-header format
//
//	:status: 200
//	content-type: text/html
//
//	body
func ParseResponse(data []byte) (*Response, error) {
	if len(data) == 0 {
		return nil, &ParseError{Message: "empty response data"}
	}

	// Detect line separator
	lineSep := detectLineSeparator(data)

	// Find header end
	headerEnd, sepLen := findHeaderEnd(data, lineSep)
	var headerSection, bodySection []byte

	if headerEnd >= 0 {
		headerSection = data[:headerEnd]
		bodyStart := headerEnd + sepLen
		if bodyStart < len(data) {
			bodySection = data[bodyStart:]
		}
	} else {
		headerSection = data
	}

	// Split into lines
	lines := splitLines(headerSection, lineSep)
	if len(lines) == 0 {
		return nil, &ParseError{Message: "no header lines found"}
	}

	// Detect format by checking first line
	firstLine := strings.TrimSpace(lines[0])
	if strings.HasPrefix(firstLine, ":") {
		// Pseudo-header format
		return parseResponsePseudoHeaderFormat(lines, bodySection)
	}

	// HTTP/1.1-style format
	return parseResponseHTTPStyleFormat(lines, bodySection)
}

// ParseResponseReader parses an HTTP/2 response from an io.Reader
func ParseResponseReader(r io.Reader) (*Response, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &ParseError{Message: "failed to read from reader: " + err.Error()}
	}
	return ParseResponse(data)
}

// parseResponseHTTPStyleFormat parses HTTP/1.1-style response with HTTP/2 version
// Format: HTTP/2 200 OK
func parseResponseHTTPStyleFormat(lines []string, body []byte) (*Response, error) {
	resp := NewResponse()

	if len(lines) == 0 {
		return nil, &ParseError{Message: "no status line"}
	}

	// Parse status line
	statusLine := strings.TrimSpace(lines[0])
	parts := strings.Fields(statusLine)

	if len(parts) < 2 {
		return nil, &ParseError{Message: "invalid status line format"}
	}

	// Check version
	version := strings.ToUpper(parts[0])
	if !strings.Contains(version, "2") && version != "H2" {
		return nil, &ParseError{Message: "not an HTTP/2 response: " + parts[0]}
	}

	// Parse status code
	statusCode, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, &ParseError{Message: "invalid status code: " + parts[1]}
	}
	resp.Status = statusCode

	// Parse headers
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx <= 0 {
			continue // Skip malformed headers
		}

		name := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		// Handle pseudo-headers
		nameLower := strings.ToLower(name)
		if nameLower == ":status" {
			code, err := strconv.Atoi(value)
			if err == nil {
				resp.Status = code
			}
			continue
		}

		// Skip connection-specific headers
		if nameLower == "connection" ||
			nameLower == "keep-alive" ||
			nameLower == "proxy-connection" ||
			nameLower == "transfer-encoding" ||
			nameLower == "upgrade" {
			continue
		}

		resp.Headers.Add(name, value)
	}

	// Set body
	if len(body) > 0 {
		resp.Body = make([]byte, len(body))
		copy(resp.Body, body)
		resp.EndStream = false
	} else {
		resp.EndStream = true
	}

	return resp, nil
}

// parseResponsePseudoHeaderFormat parses pseudo-header format response
// Format: :status: 200, etc.
func parseResponsePseudoHeaderFormat(lines []string, body []byte) (*Response, error) {
	resp := NewResponse()
	statusFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Find colon (skip first char if it's : for pseudo-headers)
		var colonIdx int
		if strings.HasPrefix(line, ":") {
			colonIdx = strings.Index(line[1:], ":")
			if colonIdx >= 0 {
				colonIdx++
			}
		} else {
			colonIdx = strings.Index(line, ":")
		}

		if colonIdx <= 0 {
			continue
		}

		name := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		// Handle pseudo-headers
		if strings.ToLower(name) == ":status" {
			code, err := strconv.Atoi(value)
			if err == nil {
				resp.Status = code
				statusFound = true
			}
			continue
		}

		// Skip connection-specific headers
		nameLower := strings.ToLower(name)
		if nameLower == "connection" ||
			nameLower == "keep-alive" ||
			nameLower == "proxy-connection" ||
			nameLower == "transfer-encoding" ||
			nameLower == "upgrade" {
			continue
		}

		resp.Headers.Add(name, value)
	}

	// Validate required pseudo-header
	if !statusFound {
		return nil, &ParseError{Message: "missing :status pseudo-header"}
	}

	// Set body
	if len(body) > 0 {
		resp.Body = make([]byte, len(body))
		copy(resp.Body, body)
		resp.EndStream = false
	} else {
		resp.EndStream = true
	}

	return resp, nil
}

// ParseError represents an HTTP/2 parsing error
type ParseError struct {
	Message string
}

func (e *ParseError) Error() string {
	return "http2 parse error: " + e.Message
}

// Helper functions

// detectLineSeparator detects the line separator used in the data
func detectLineSeparator(data []byte) string {
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\r' && data[i+1] == '\n' {
			return "\r\n"
		}
		if data[i] == '\n' {
			return "\n"
		}
	}
	return "\r\n" // Default
}

// findHeaderEnd finds the end of headers and returns position and separator length
func findHeaderEnd(data []byte, lineSep string) (int, int) {
	if lineSep == "\r\n" {
		// Look for \r\n\r\n
		for i := 0; i < len(data)-3; i++ {
			if data[i] == '\r' && data[i+1] == '\n' &&
				data[i+2] == '\r' && data[i+3] == '\n' {
				return i, 4
			}
		}
	}

	// Look for \n\n (fallback)
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\n' && data[i+1] == '\n' {
			return i, 2
		}
	}

	return -1, 0
}

// splitLines splits data into lines based on separator
func splitLines(data []byte, lineSep string) []string {
	str := string(data)
	var lines []string

	if lineSep == "\r\n" {
		lines = strings.Split(str, "\r\n")
	} else {
		lines = strings.Split(str, "\n")
	}

	// Trim empty trailing lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}
