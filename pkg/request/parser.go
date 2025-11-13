package request

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/errors"
	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// Parse parses raw HTTP request data with fault tolerance
func Parse(data []byte) (*Request, error) {
	if len(data) == 0 {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"empty request data", "parse", data)
	}

	req := NewRequest()
	req.Raw = make([]byte, len(data))
	copy(req.Raw, data)

	// Split request into lines
	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"no request line found", "parse", data)
	}

	// Parse request line (Method URL Version)
	requestLine := scanner.Text()
	if err := req.parseRequestLine(requestLine); err != nil {
		return nil, err
	}

	// Parse headers
	headerData := &bytes.Buffer{}
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			break // End of headers
		}
		headerData.WriteString(line)
		headerData.WriteString("\r\n")
	}

	if headerData.Len() > 0 {
		parsedHeaders, err := headers.ParseHeaders(headerData.Bytes())
		if err != nil {
			// Continue with empty headers on parse error (fault tolerance)
			req.Headers = headers.NewOrderedHeaders()
		} else {
			req.Headers = parsedHeaders
		}
	}

	// Read body (everything after headers)
	bodyData := &bytes.Buffer{}
	for scanner.Scan() {
		bodyData.WriteString(scanner.Text())
		bodyData.WriteString("\r\n")
	}

	// Remove trailing CRLF from body if present
	bodyBytes := bodyData.Bytes()
	if len(bodyBytes) > 2 && string(bodyBytes[len(bodyBytes)-2:]) == "\r\n" {
		bodyBytes = bodyBytes[:len(bodyBytes)-2]
	}
	req.Body = bodyBytes

	// Auto-parse Transfer-Encoding header
	req.parseTransferEncoding()

	// Auto-parse query parameters from URL
	req.ParseQueryParams()

	// Auto-parse cookies from Cookie header
	req.ParseCookies()

	return req, nil
}

// parseTransferEncoding parses Transfer-Encoding header
func (r *Request) parseTransferEncoding() {
	teHeader := r.Headers.Get("Transfer-Encoding")
	if teHeader == "" {
		r.TransferEncoding = []string{}
		return
	}

	// Split by comma
	parts := strings.Split(teHeader, ",")
	encodings := make([]string, 0, len(parts))

	for _, part := range parts {
		encoding := strings.TrimSpace(part)
		if encoding != "" {
			encodings = append(encodings, encoding)
		}
	}

	r.TransferEncoding = encodings

	// Check if body is chunked
	for _, enc := range encodings {
		if strings.ToLower(enc) == "chunked" {
			r.IsBodyChunked = true
			break
		}
	}
}

// parseRequestLine parses the HTTP request line with fault tolerance
func (r *Request) parseRequestLine(line string) error {
	parts := strings.Fields(line)

	if len(parts) < 2 {
		return errors.NewError(errors.ErrorTypeInvalidFormat,
			"invalid request line format", "parseRequestLine", []byte(line))
	}

	// Method
	r.Method = strings.ToUpper(parts[0])
	if r.Method == "" {
		return errors.NewError(errors.ErrorTypeInvalidMethod,
			"empty HTTP method", "parseRequestLine", []byte(line))
	}

	// URL/Path
	r.URL = parts[1]
	if r.URL == "" {
		return errors.NewError(errors.ErrorTypeInvalidURL,
			"empty URL/path", "parseRequestLine", []byte(line))
	}

	// Version (optional, default to HTTP/1.1)
	if len(parts) >= 3 {
		r.Version = parts[2]
	} else {
		r.Version = "HTTP/1.1" // Default version for fault tolerance
	}

	// Validate version format
	if !strings.HasPrefix(strings.ToUpper(r.Version), "HTTP/") {
		// Keep the invalid version but mark as fault tolerance
		r.Version = "HTTP/1.1"
	}

	return nil
}
