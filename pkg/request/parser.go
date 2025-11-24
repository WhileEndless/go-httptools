package request

import (
	"io"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/compression"
	"github.com/WhileEndless/go-httptools/pkg/errors"
	"github.com/WhileEndless/go-httptools/pkg/headers"
)

// Parse parses raw HTTP request data with fault tolerance
// Preserves original header formatting and line endings
func Parse(data []byte) (*Request, error) {
	return parse(data)
}

// ParseReader parses an HTTP request from an io.Reader
// Reads all data from the reader and parses it
// Suitable for integration with streaming sources like buffers or file readers
func ParseReader(r io.Reader) (*Request, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"failed to read from reader: "+err.Error(), "parseReader", nil)
	}
	return parse(data)
}

// parse is the internal implementation for parsing HTTP request data
func parse(data []byte) (*Request, error) {
	if len(data) == 0 {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"empty request data", "parse", data)
	}

	req := NewRequest()
	req.Raw = make([]byte, len(data))
	copy(req.Raw, data)

	// Find first line ending to extract request line and detect line separator
	requestLineEnd := 0
	for requestLineEnd < len(data) && data[requestLineEnd] != '\n' && data[requestLineEnd] != '\r' {
		requestLineEnd++
	}

	if requestLineEnd == 0 {
		return nil, errors.NewError(errors.ErrorTypeInvalidFormat,
			"no request line found", "parse", data)
	}

	// Detect line separator from first line
	if requestLineEnd < len(data) {
		if data[requestLineEnd] == '\r' && requestLineEnd+1 < len(data) && data[requestLineEnd+1] == '\n' {
			req.LineSeparator = "\r\n"
		} else if data[requestLineEnd] == '\n' {
			req.LineSeparator = "\n"
		} else if data[requestLineEnd] == '\r' {
			req.LineSeparator = "\r"
		}
	}

	// Parse request line (Method URL Version)
	requestLine := string(data[:requestLineEnd])
	if err := req.parseRequestLine(requestLine); err != nil {
		return nil, err
	}

	// Skip past request line and its line ending
	headerStart := requestLineEnd
	if headerStart < len(data) && data[headerStart] == '\r' {
		headerStart++
	}
	if headerStart < len(data) && data[headerStart] == '\n' {
		headerStart++
	}

	// Find header section end
	headerEnd := findHeaderEndIndex(data)
	if headerEnd < 0 {
		headerEnd = len(data)
	}

	// Calculate header data end position (include last line ending)
	// findHeaderEndIndex returns the start of \r\n\r\n or \n\n
	// We need to include the first \r\n or \n (the last header's line ending)
	headerDataEnd := headerEnd
	if headerEnd < len(data) {
		// Include the last header's line ending
		if data[headerEnd] == '\r' {
			headerDataEnd++
			if headerDataEnd < len(data) && data[headerDataEnd] == '\n' {
				headerDataEnd++
			}
		} else if data[headerEnd] == '\n' {
			headerDataEnd++
		}
	}

	// Extract header section with original line endings preserved
	if headerStart < headerDataEnd {
		headerData := data[headerStart:headerDataEnd]
		parsedHeaders, err := headers.ParseHeaders(headerData)
		if err != nil {
			req.Headers = headers.NewOrderedHeaders()
		} else {
			req.Headers = parsedHeaders
		}
	}

	// Read body (everything after headers)
	var bodyBytes []byte
	if headerEnd >= 0 && headerEnd < len(data) {
		separatorLen := getHeaderSeparatorLength(data, headerEnd)
		bodyStart := headerEnd + separatorLen
		if bodyStart < len(data) {
			bodyBytes = data[bodyStart:]
		} else {
			bodyBytes = []byte{}
		}
	} else {
		bodyBytes = []byte{}
	}

	// Store raw body
	req.RawBody = bodyBytes

	// Detect compression - first try header, then magic bytes
	contentEncoding := req.GetContentEncoding()
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
	req.DetectedCompression = compressionType

	// Decompress if compression was detected
	if compressionType != compression.CompressionNone {
		decompressed, err := compression.Decompress(bodyBytes, compressionType)
		if err != nil {
			// On decompression error, keep raw body (fault tolerance)
			req.Body = bodyBytes
			req.Compressed = false
			req.DetectedCompression = compression.CompressionNone
		} else {
			req.Body = decompressed
			req.Compressed = true
		}
	} else {
		req.Body = bodyBytes
		req.Compressed = false
	}

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

// findHeaderEndIndex finds the index of the header end marker (\r\n\r\n or \n\n)
// Returns both the index and the length of the separator (4 for CRLF, 2 for LF)
func findHeaderEndIndex(data []byte) int {
	// First try to find CRLF separator (\r\n\r\n)
	for i := 0; i < len(data)-3; i++ {
		if data[i] == '\r' && data[i+1] == '\n' &&
			data[i+2] == '\r' && data[i+3] == '\n' {
			return i
		}
	}

	// Fallback: try to find LF separator (\n\n) for fault tolerance
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\n' && data[i+1] == '\n' {
			return i
		}
	}

	return -1
}

// getHeaderSeparatorLength returns the length of the header separator at the given position
func getHeaderSeparatorLength(data []byte, pos int) int {
	// Check if it's CRLF\CRLF (4 bytes)
	if pos+3 < len(data) &&
		data[pos] == '\r' && data[pos+1] == '\n' &&
		data[pos+2] == '\r' && data[pos+3] == '\n' {
		return 4
	}

	// Check if it's LF\LF (2 bytes)
	if pos+1 < len(data) && data[pos] == '\n' && data[pos+1] == '\n' {
		return 2
	}

	// Default to CRLF\CRLF
	return 4
}
