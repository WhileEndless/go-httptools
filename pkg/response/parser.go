package response

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/compression"
	"github.com/WhileEndless/go-httptools/pkg/errors"
	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// Parse parses raw HTTP response data with fault tolerance and automatic decompression
func Parse(data []byte) (*Response, error) {
	if len(data) == 0 {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"empty response data", "parse", data)
	}

	resp := NewResponse()
	resp.Raw = make([]byte, len(data))
	copy(resp.Raw, data)

	// Split response into lines
	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"no status line found", "parse", data)
	}

	// Parse status line (Version StatusCode StatusText)
	statusLine := scanner.Text()
	if err := resp.parseStatusLine(statusLine); err != nil {
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
			resp.Headers = headers.NewOrderedHeaders()
		} else {
			resp.Headers = parsedHeaders
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

	// Store raw body and attempt decompression
	resp.RawBody = bodyBytes

	// Detect and handle compression
	contentEncoding := resp.GetContentEncoding()
	if contentEncoding != "" {
		compressionType := compression.DetectCompression(contentEncoding)
		if compressionType != compression.CompressionNone {
			decompressed, err := compression.Decompress(bodyBytes, compressionType)
			if err != nil {
				// On decompression error, keep raw body (fault tolerance)
				resp.Body = bodyBytes
				resp.Compressed = false
			} else {
				resp.Body = decompressed
				resp.Compressed = true
			}
		} else {
			resp.Body = bodyBytes
			resp.Compressed = false
		}
	} else {
		resp.Body = bodyBytes
		resp.Compressed = false
	}

	return resp, nil
}

// parseStatusLine parses the HTTP status line with fault tolerance
func (r *Response) parseStatusLine(line string) error {
	parts := strings.Fields(line)

	if len(parts) < 2 {
		return errors.NewError(errors.ErrorTypeInvalidFormat,
			"invalid status line format", "parseStatusLine", []byte(line))
	}

	// Version
	r.Version = parts[0]
	if !strings.HasPrefix(strings.ToUpper(r.Version), "HTTP/") {
		// Keep the invalid version but set default for fault tolerance
		r.Version = "HTTP/1.1"
	}

	// Status Code
	statusCodeStr := parts[1]
	statusCode, err := strconv.Atoi(statusCodeStr)
	if err != nil {
		return errors.NewError(errors.ErrorTypeInvalidStatusCode,
			"invalid status code: "+statusCodeStr, "parseStatusLine", []byte(line))
	}
	r.StatusCode = statusCode

	// Status Text (optional, may contain spaces)
	if len(parts) >= 3 {
		r.StatusText = strings.Join(parts[2:], " ")
	} else {
		// Provide default status text based on status code
		r.StatusText = getDefaultStatusText(statusCode)
	}

	return nil
}

// getDefaultStatusText provides default status text for common HTTP status codes
func getDefaultStatusText(statusCode int) string {
	switch statusCode {
	case 100:
		return "Continue"
	case 101:
		return "Switching Protocols"
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 304:
		return "Not Modified"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	default:
		return "Unknown"
	}
}
