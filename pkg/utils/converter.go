package utils

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
)

// ToStandardRequest converts our Request to standard http.Request
func ToStandardRequest(req *request.Request) (*http.Request, error) {
	// Create standard request
	httpReq, err := http.NewRequest(req.Method, req.URL, strings.NewReader(string(req.Body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create standard request: %w", err)
	}

	// Copy headers
	for _, header := range req.Headers.All() {
		httpReq.Header.Set(header.Name, header.Value)
	}

	return httpReq, nil
}

// FromStandardRequest converts standard http.Request to our Request
func FromStandardRequest(httpReq *http.Request) *request.Request {
	req := request.NewRequest()
	req.Method = httpReq.Method
	req.URL = httpReq.URL.String()
	req.Version = fmt.Sprintf("HTTP/%d.%d", httpReq.ProtoMajor, httpReq.ProtoMinor)

	// Copy headers in order (best effort to preserve order)
	for name, values := range httpReq.Header {
		if len(values) > 0 {
			req.Headers.Set(name, values[0]) // Take first value
		}
	}

	// Read body if present
	if httpReq.Body != nil {
		// Note: This consumes the body, so use carefully
		bodyData := make([]byte, httpReq.ContentLength)
		httpReq.Body.Read(bodyData)
		req.Body = bodyData
	}

	return req
}

// ToStandardResponse converts our Response to standard http.Response
func ToStandardResponse(resp *response.Response) *http.Response {
	httpResp := &http.Response{
		Status:     fmt.Sprintf("%d %s", resp.StatusCode, resp.StatusText),
		StatusCode: resp.StatusCode,
		Proto:      resp.Version,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(resp.Body))),
	}

	// Parse version
	if strings.HasPrefix(resp.Version, "HTTP/") {
		version := strings.TrimPrefix(resp.Version, "HTTP/")
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			if parts[0] == "1" && parts[1] == "1" {
				httpResp.ProtoMajor = 1
				httpResp.ProtoMinor = 1
			} else if parts[0] == "2" {
				httpResp.ProtoMajor = 2
				httpResp.ProtoMinor = 0
			}
		}
	}

	// Copy headers
	for _, header := range resp.Headers.All() {
		httpResp.Header.Set(header.Name, header.Value)
	}

	return httpResp
}

// FromStandardResponse converts standard http.Response to our Response
func FromStandardResponse(httpResp *http.Response) *response.Response {
	resp := response.NewResponse()
	resp.StatusCode = httpResp.StatusCode
	resp.StatusText = httpResp.Status[4:] // Remove "XXX " prefix
	resp.Version = fmt.Sprintf("HTTP/%d.%d", httpResp.ProtoMajor, httpResp.ProtoMinor)

	// Copy headers in order (best effort to preserve order)
	for name, values := range httpResp.Header {
		if len(values) > 0 {
			resp.Headers.Set(name, values[0]) // Take first value
		}
	}

	return resp
}
