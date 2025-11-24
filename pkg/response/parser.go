package response

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/chunked"
	"github.com/WhileEndless/go-httptools/pkg/compression"
	"github.com/WhileEndless/go-httptools/pkg/cookies"
	"github.com/WhileEndless/go-httptools/pkg/errors"
	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// ParseOptions contains options for parsing HTTP responses
type ParseOptions struct {
	// AutoDecodeChunked automatically decodes chunked transfer encoding
	// When true, the Body field will contain the decoded content
	// When false (default), chunked bodies remain encoded and IsBodyChunked=true
	AutoDecodeChunked bool

	// PreserveChunkedTrailers stores trailers from chunked encoding as headers
	// Only effective when AutoDecodeChunked is true
	PreserveChunkedTrailers bool
}

// Parse parses raw HTTP response data with fault tolerance and automatic decompression
// Uses default options (no automatic chunked decoding)
func Parse(data []byte) (*Response, error) {
	return ParseWithOptions(data, ParseOptions{})
}

// ParseWithOptions parses raw HTTP response data with custom options
// Preserves original header formatting and line endings
func ParseWithOptions(data []byte, opts ParseOptions) (*Response, error) {
	if len(data) == 0 {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"empty response data", "parse", data)
	}

	resp := NewResponse()
	resp.Raw = make([]byte, len(data))
	copy(resp.Raw, data)

	// Find first line ending to extract status line and detect line separator
	statusLineEnd := 0
	for statusLineEnd < len(data) && data[statusLineEnd] != '\n' && data[statusLineEnd] != '\r' {
		statusLineEnd++
	}

	if statusLineEnd == 0 {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"no status line found", "parse", data)
	}

	// Detect line separator from first line
	if statusLineEnd < len(data) {
		if data[statusLineEnd] == '\r' && statusLineEnd+1 < len(data) && data[statusLineEnd+1] == '\n' {
			resp.LineSeparator = "\r\n"
		} else if data[statusLineEnd] == '\n' {
			resp.LineSeparator = "\n"
		} else if data[statusLineEnd] == '\r' {
			resp.LineSeparator = "\r"
		}
	}

	// Parse status line (Version StatusCode StatusText)
	statusLine := string(data[:statusLineEnd])
	if err := resp.parseStatusLine(statusLine); err != nil {
		return nil, err
	}

	// Skip past status line and its line ending
	headerStart := statusLineEnd
	if headerStart < len(data) && data[headerStart] == '\r' {
		headerStart++
	}
	if headerStart < len(data) && data[headerStart] == '\n' {
		headerStart++
	}

	// Find the end of headers (double line break)
	headerEndIdx := findHeaderEndIndex(data)
	if headerEndIdx == -1 {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"no header end found", "parse", data)
	}

	// Calculate header data end position (include last line ending)
	headerDataEnd := headerEndIdx
	if headerEndIdx < len(data) {
		if data[headerEndIdx] == '\r' {
			headerDataEnd++
			if headerDataEnd < len(data) && data[headerDataEnd] == '\n' {
				headerDataEnd++
			}
		} else if data[headerEndIdx] == '\n' {
			headerDataEnd++
		}
	}

	// Extract Set-Cookie headers before parsing (for multi-value support)
	setCookieHeaders := []string{}
	if headerStart < headerDataEnd {
		headerSection := data[headerStart:headerDataEnd]
		setCookieHeaders = extractSetCookieHeaders(headerSection)
	}

	// Parse headers with original formatting preserved
	if headerStart < headerDataEnd {
		headerData := data[headerStart:headerDataEnd]
		parsedHeaders, err := headers.ParseHeaders(headerData)
		if err != nil {
			resp.Headers = headers.NewOrderedHeaders()
		} else {
			resp.Headers = parsedHeaders
		}
	}

	// Parse Set-Cookie headers collected separately
	for _, setCookieValue := range setCookieHeaders {
		cookie := cookies.ParseSetCookie(setCookieValue)
		resp.SetCookies = append(resp.SetCookies, cookie)
	}

	// Get body bytes
	bodyStart := findHeaderEnd(data)
	if bodyStart == -1 {
		bodyStart = len(data)
	}
	bodyBytes := data[bodyStart:]

	// Store raw body and attempt decompression
	resp.RawBody = bodyBytes

	// Detect compression - first try header, then magic bytes
	contentEncoding := resp.GetContentEncoding()
	compressionType := compression.CompressionNone

	if contentEncoding != "" {
		// Try header-based detection first
		compressionType = compression.DetectCompression(contentEncoding)
	}

	// If header didn't indicate compression, try magic byte detection
	if compressionType == compression.CompressionNone && len(bodyBytes) > 0 {
		compressionType = compression.DetectByMagicBytes(bodyBytes)
	}

	// Store detected compression type
	resp.DetectedCompression = compressionType

	// Decompress if compression was detected
	if compressionType != compression.CompressionNone {
		decompressed, err := compression.Decompress(bodyBytes, compressionType)
		if err != nil {
			// On decompression error, keep raw body (fault tolerance)
			resp.Body = bodyBytes
			resp.Compressed = false
			resp.DetectedCompression = compression.CompressionNone
		} else {
			resp.Body = decompressed
			resp.Compressed = true
		}
	} else {
		resp.Body = bodyBytes
		resp.Compressed = false
	}

	// Auto-parse Transfer-Encoding header
	resp.parseTransferEncoding()

	// Auto-decode chunked transfer encoding if requested
	if opts.AutoDecodeChunked && resp.IsBodyChunked {
		decodedBody, trailers := chunked.Decode(resp.Body)

		// Store original chunked body in RawBody
		resp.RawBody = resp.Body

		// Update body with decoded version
		resp.Body = decodedBody
		resp.IsBodyChunked = false

		// Preserve trailers as headers if requested
		if opts.PreserveChunkedTrailers && len(trailers) > 0 {
			for name, value := range trailers {
				resp.Headers.Set(name, value)
			}
		}

		// Remove Transfer-Encoding: chunked header
		resp.Headers.Del("Transfer-Encoding")
		resp.TransferEncoding = []string{}

		// Add Content-Length header with decoded body size
		if len(decodedBody) > 0 {
			resp.Headers.Set("Content-Length", strconv.Itoa(len(decodedBody)))
		}
	}

	// Set-Cookie headers already parsed above during header parsing

	return resp, nil
}

// parseTransferEncoding parses Transfer-Encoding header
func (r *Response) parseTransferEncoding() {
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

// findHeaderEnd finds the position after the double line break that separates headers from body
// Returns the index where body starts (after \r\n\r\n or \n\n)
func findHeaderEnd(data []byte) int {
	// Try CRLF first (\r\n\r\n)
	idx := bytes.Index(data, []byte("\r\n\r\n"))
	if idx != -1 {
		return idx + 4
	}

	// Try Unix line endings (\n\n)
	idx = bytes.Index(data, []byte("\n\n"))
	if idx != -1 {
		return idx + 2
	}

	return -1
}

// findHeaderEndIndex finds the start of the header-body separator
// Returns the index of the first \r or \n of \r\n\r\n or \n\n
func findHeaderEndIndex(data []byte) int {
	// Try CRLF first (\r\n\r\n)
	idx := bytes.Index(data, []byte("\r\n\r\n"))
	if idx != -1 {
		return idx
	}

	// Try Unix line endings (\n\n)
	idx = bytes.Index(data, []byte("\n\n"))
	if idx != -1 {
		return idx
	}

	return -1
}

// extractSetCookieHeaders extracts Set-Cookie header values from header section
func extractSetCookieHeaders(headerData []byte) []string {
	var setCookies []string

	i := 0
	for i < len(headerData) {
		// Find end of current line
		lineStart := i
		lineEnd := i
		for lineEnd < len(headerData) && headerData[lineEnd] != '\n' && headerData[lineEnd] != '\r' {
			lineEnd++
		}

		line := string(headerData[lineStart:lineEnd])

		// Check if this is a Set-Cookie header (case-insensitive)
		if len(line) > 11 {
			lowerLine := strings.ToLower(line)
			if strings.HasPrefix(lowerLine, "set-cookie:") {
				colonPos := strings.Index(line, ":")
				if colonPos != -1 {
					value := strings.TrimSpace(line[colonPos+1:])
					setCookies = append(setCookies, value)
				}
			}
		}

		// Skip past line ending
		if lineEnd < len(headerData) && headerData[lineEnd] == '\r' {
			lineEnd++
		}
		if lineEnd < len(headerData) && headerData[lineEnd] == '\n' {
			lineEnd++
		}
		i = lineEnd
	}

	return setCookies
}
