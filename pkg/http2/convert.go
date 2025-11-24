package http2

import (
	"strconv"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
)

// FromHTTP1Request converts an HTTP/1.1 request to HTTP/2 format
// Header order is preserved
func FromHTTP1Request(req *request.Request) *Request {
	h2req := NewRequest()

	// Set pseudo-headers from HTTP/1.1 request
	h2req.Method = req.Method
	h2req.Path = req.URL

	// Determine scheme from URL or default to https
	if strings.HasPrefix(strings.ToLower(req.URL), "https://") {
		h2req.Scheme = "https"
	} else if strings.HasPrefix(strings.ToLower(req.URL), "http://") {
		h2req.Scheme = "http"
	} else {
		h2req.Scheme = "https" // Default to https for HTTP/2
	}

	// Set authority from Host header
	host := strings.TrimSpace(req.Headers.Get("Host"))
	if host != "" {
		h2req.Authority = host
	}

	// Copy headers (excluding pseudo-header equivalents)
	for _, hdr := range req.Headers.All() {
		nameLower := strings.ToLower(hdr.Name)

		// Skip headers that are pseudo-headers in HTTP/2
		if nameLower == "host" {
			continue // Already in :authority
		}

		// Skip connection-specific headers (HTTP/2 doesn't use these)
		if nameLower == "connection" ||
			nameLower == "keep-alive" ||
			nameLower == "proxy-connection" ||
			nameLower == "transfer-encoding" ||
			nameLower == "upgrade" {
			continue
		}

		// Trim value whitespace (HTTP/1.1 preserves original whitespace but HTTP/2 doesn't)
		h2req.Headers.Add(hdr.Name, strings.TrimSpace(hdr.Value))
	}

	// Copy body
	if len(req.Body) > 0 {
		h2req.Body = make([]byte, len(req.Body))
		copy(h2req.Body, req.Body)
		h2req.EndStream = false
	} else {
		h2req.EndStream = true
	}

	return h2req
}

// ToHTTP1Request converts an HTTP/2 request to HTTP/1.1 format
// Header order is preserved
func ToHTTP1Request(h2req *Request) *request.Request {
	req := request.NewRequest()

	// Set request line from pseudo-headers
	req.Method = h2req.Method
	req.URL = h2req.Path
	req.Version = "HTTP/1.1"

	// Set Host header from :authority
	if h2req.Authority != "" {
		req.Headers.Set("Host", h2req.Authority)
	}

	// Copy headers (maintaining order)
	for _, hdr := range h2req.Headers.All() {
		req.Headers.Add(hdr.Name, hdr.Value)
	}

	// Copy body
	if len(h2req.Body) > 0 {
		req.Body = make([]byte, len(h2req.Body))
		copy(req.Body, h2req.Body)
	}

	return req
}

// FromHTTP1Response converts an HTTP/1.1 response to HTTP/2 format
// Header order is preserved
func FromHTTP1Response(resp *response.Response) *Response {
	h2resp := NewResponse()

	// Set pseudo-header from HTTP/1.1 response
	h2resp.Status = resp.StatusCode

	// Copy headers (excluding connection-specific ones)
	for _, hdr := range resp.Headers.All() {
		nameLower := strings.ToLower(hdr.Name)

		// Skip connection-specific headers
		if nameLower == "connection" ||
			nameLower == "keep-alive" ||
			nameLower == "proxy-connection" ||
			nameLower == "transfer-encoding" ||
			nameLower == "upgrade" {
			continue
		}

		// Trim value whitespace (HTTP/1.1 preserves original whitespace but HTTP/2 doesn't)
		h2resp.Headers.Add(hdr.Name, strings.TrimSpace(hdr.Value))
	}

	// Copy body
	if len(resp.Body) > 0 {
		h2resp.Body = make([]byte, len(resp.Body))
		copy(h2resp.Body, resp.Body)
		h2resp.EndStream = false
	} else {
		h2resp.EndStream = true
	}

	return h2resp
}

// ToHTTP1Response converts an HTTP/2 response to HTTP/1.1 format
// Header order is preserved
func ToHTTP1Response(h2resp *Response) *response.Response {
	resp := response.NewResponse()

	// Set status line from pseudo-header
	resp.Version = "HTTP/1.1"
	resp.StatusCode = h2resp.Status
	resp.StatusText = h2resp.GetStatusText()

	// Copy headers (maintaining order)
	for _, hdr := range h2resp.Headers.All() {
		resp.Headers.Add(hdr.Name, hdr.Value)
	}

	// Copy body
	if len(h2resp.Body) > 0 {
		resp.Body = make([]byte, len(h2resp.Body))
		copy(resp.Body, h2resp.Body)
	}

	return resp
}

// ParseHeaderBlock parses a slice of header fields into an HTTP/2 request
// Useful for parsing HPACK-decoded headers
func ParseRequestHeaders(fields []HeaderField) *Request {
	req := NewRequest()

	for _, f := range fields {
		switch f.Name {
		case ":method":
			req.Method = f.Value
		case ":scheme":
			req.Scheme = f.Value
		case ":authority":
			req.Authority = f.Value
		case ":path":
			req.Path = f.Value
		default:
			if f.Sensitive {
				req.Headers.AddSensitive(f.Name, f.Value)
			} else {
				req.Headers.Add(f.Name, f.Value)
			}
		}
	}

	return req
}

// ParseResponseHeaders parses a slice of header fields into an HTTP/2 response
func ParseResponseHeaders(fields []HeaderField) *Response {
	resp := NewResponse()

	for _, f := range fields {
		if f.Name == ":status" {
			status, err := strconv.Atoi(f.Value)
			if err == nil {
				resp.Status = status
			}
		} else {
			if f.Sensitive {
				resp.Headers.AddSensitive(f.Name, f.Value)
			} else {
				resp.Headers.Add(f.Name, f.Value)
			}
		}
	}

	return resp
}

// BuildRequestLine returns an HTTP/1.1 style request line for debugging
func (r *Request) BuildRequestLine() string {
	return r.Method + " " + r.Path + " HTTP/2"
}

// BuildStatusLine returns an HTTP/1.1 style status line for debugging
func (r *Response) BuildStatusLine() string {
	return "HTTP/2 " + strconv.Itoa(r.Status) + " " + r.GetStatusText()
}
