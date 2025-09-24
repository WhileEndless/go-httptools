package utils

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/headers"
	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
)

// ValidationResult contains validation results
type ValidationResult struct {
	Valid    bool
	Warnings []string
	Errors   []string
}

// ValidateRequest validates a request and returns warnings/errors
func ValidateRequest(req *request.Request) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Warnings: make([]string, 0),
		Errors:   make([]string, 0),
	}

	// Validate HTTP method
	if req.Method == "" {
		result.Errors = append(result.Errors, "HTTP method is empty")
		result.Valid = false
	} else {
		validMethods := []string{"GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}
		isValid := false
		for _, method := range validMethods {
			if req.Method == method {
				isValid = true
				break
			}
		}
		if !isValid {
			result.Warnings = append(result.Warnings, "Non-standard HTTP method: "+req.Method)
		}
	}

	// Validate URL
	if req.URL == "" {
		result.Errors = append(result.Errors, "URL is empty")
		result.Valid = false
	} else {
		// Try to parse URL
		if strings.HasPrefix(req.URL, "http://") || strings.HasPrefix(req.URL, "https://") {
			if _, err := url.Parse(req.URL); err != nil {
				result.Warnings = append(result.Warnings, "Invalid URL format: "+err.Error())
			}
		}
	}

	// Validate HTTP version
	if req.Version == "" {
		result.Warnings = append(result.Warnings, "HTTP version is empty")
	} else if !strings.HasPrefix(strings.ToUpper(req.Version), "HTTP/") {
		result.Warnings = append(result.Warnings, "Invalid HTTP version format: "+req.Version)
	}

	// Validate headers
	validateHeaders(req.Headers.All(), result)

	// Validate Content-Length vs body size
	if contentLength := req.GetContentLength(); contentLength != "" {
		if length, err := strconv.Atoi(contentLength); err != nil {
			result.Warnings = append(result.Warnings, "Invalid Content-Length header: "+contentLength)
		} else if length != len(req.Body) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Content-Length mismatch: header says %d, body is %d bytes", length, len(req.Body)))
		}
	}

	// Check for body with GET/HEAD methods
	if (req.Method == "GET" || req.Method == "HEAD") && len(req.Body) > 0 {
		result.Warnings = append(result.Warnings, req.Method+" request with body (non-standard)")
	}

	return result
}

// ValidateResponse validates a response and returns warnings/errors
func ValidateResponse(resp *response.Response) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Warnings: make([]string, 0),
		Errors:   make([]string, 0),
	}

	// Validate status code
	if resp.StatusCode < 100 || resp.StatusCode > 599 {
		result.Errors = append(result.Errors, "Invalid HTTP status code: "+strconv.Itoa(resp.StatusCode))
		result.Valid = false
	}

	// Validate HTTP version
	if resp.Version == "" {
		result.Warnings = append(result.Warnings, "HTTP version is empty")
	} else if !strings.HasPrefix(strings.ToUpper(resp.Version), "HTTP/") {
		result.Warnings = append(result.Warnings, "Invalid HTTP version format: "+resp.Version)
	}

	// Validate headers
	validateHeaders(resp.Headers.All(), result)

	// Validate Content-Length vs body size
	contentLength := resp.GetContentLength()
	if contentLength > 0 {
		bodySize := len(resp.RawBody) // Use raw body for accuracy
		if contentLength != bodySize {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Content-Length mismatch: header says %d, body is %d bytes", contentLength, bodySize))
		}
	}

	// Check compression consistency
	contentEncoding := resp.GetContentEncoding()
	if contentEncoding != "" && !resp.Compressed {
		result.Warnings = append(result.Warnings, "Content-Encoding header present but body not compressed")
	} else if contentEncoding == "" && resp.Compressed {
		result.Warnings = append(result.Warnings, "Body is compressed but no Content-Encoding header")
	}

	return result
}

// validateHeaders validates common header issues
func validateHeaders(headerList []headers.Header, result *ValidationResult) {
	headerNames := make(map[string]int)

	for _, header := range headerList {
		// Check for duplicate headers (case-insensitive)
		lowerName := strings.ToLower(header.Name)
		headerNames[lowerName]++
		if headerNames[lowerName] > 1 {
			result.Warnings = append(result.Warnings, "Duplicate header: "+header.Name)
		}

		// Check for empty header name
		if strings.TrimSpace(header.Name) == "" {
			result.Warnings = append(result.Warnings, "Empty header name")
		}

		// Check for headers with only whitespace
		if strings.TrimSpace(header.Value) == "" && header.Value != "" {
			result.Warnings = append(result.Warnings, "Header with only whitespace value: "+header.Name)
		}

		// Check for potentially malicious headers
		if strings.Contains(header.Name, "\n") || strings.Contains(header.Name, "\r") {
			result.Errors = append(result.Errors, "Header name contains newline characters: "+header.Name)
			result.Valid = false
		}
		if strings.Contains(header.Value, "\n") || strings.Contains(header.Value, "\r") {
			result.Errors = append(result.Errors, "Header value contains newline characters: "+header.Name)
			result.Valid = false
		}
	}
}
