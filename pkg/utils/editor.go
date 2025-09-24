package utils

import (
	"net/url"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
)

// RequestEditor provides Burp Suite-like editing capabilities for HTTP requests
type RequestEditor struct {
	req *request.Request
}

// NewRequestEditor creates a new request editor
func NewRequestEditor(req *request.Request) *RequestEditor {
	return &RequestEditor{req: req.Clone()}
}

// GetRequest returns the current request state
func (e *RequestEditor) GetRequest() *request.Request {
	return e.req
}

// SetMethod changes the HTTP method
func (e *RequestEditor) SetMethod(method string) *RequestEditor {
	e.req.Method = strings.ToUpper(method)
	return e
}

// SetURL changes the request URL/path
func (e *RequestEditor) SetURL(urlPath string) *RequestEditor {
	e.req.URL = urlPath
	return e
}

// SetVersion changes the HTTP version
func (e *RequestEditor) SetVersion(version string) *RequestEditor {
	e.req.Version = version
	return e
}

// AddHeader adds a new header (preserves order)
func (e *RequestEditor) AddHeader(name, value string) *RequestEditor {
	e.req.Headers.Set(name, value)
	return e
}

// RemoveHeader removes a header
func (e *RequestEditor) RemoveHeader(name string) *RequestEditor {
	e.req.Headers.Del(name)
	return e
}

// UpdateHeader updates an existing header value
func (e *RequestEditor) UpdateHeader(name, value string) *RequestEditor {
	e.req.Headers.Set(name, value)
	return e
}

// SetBody sets the request body and updates Content-Length
func (e *RequestEditor) SetBody(body []byte) *RequestEditor {
	e.req.SetBody(body)
	return e
}

// SetBodyString sets the request body from string
func (e *RequestEditor) SetBodyString(body string) *RequestEditor {
	e.req.SetBody([]byte(body))
	return e
}

// AddQueryParam adds a query parameter to the URL
func (e *RequestEditor) AddQueryParam(key, value string) *RequestEditor {
	u, err := url.Parse(e.req.URL)
	if err != nil {
		return e // Skip on parse error
	}

	q := u.Query()
	q.Add(key, value)
	u.RawQuery = q.Encode()
	e.req.URL = u.String()
	return e
}

// SetQueryParam sets a query parameter (replaces existing)
func (e *RequestEditor) SetQueryParam(key, value string) *RequestEditor {
	u, err := url.Parse(e.req.URL)
	if err != nil {
		return e // Skip on parse error
	}

	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	e.req.URL = u.String()
	return e
}

// RemoveQueryParam removes a query parameter
func (e *RequestEditor) RemoveQueryParam(key string) *RequestEditor {
	u, err := url.Parse(e.req.URL)
	if err != nil {
		return e // Skip on parse error
	}

	q := u.Query()
	q.Del(key)
	u.RawQuery = q.Encode()
	e.req.URL = u.String()
	return e
}

// ResponseEditor provides Burp Suite-like editing capabilities for HTTP responses
type ResponseEditor struct {
	resp *response.Response
}

// NewResponseEditor creates a new response editor
func NewResponseEditor(resp *response.Response) *ResponseEditor {
	return &ResponseEditor{resp: resp.Clone()}
}

// GetResponse returns the current response state
func (e *ResponseEditor) GetResponse() *response.Response {
	return e.resp
}

// SetStatusCode changes the HTTP status code
func (e *ResponseEditor) SetStatusCode(code int) *ResponseEditor {
	e.resp.StatusCode = code
	return e
}

// SetStatusText changes the status text
func (e *ResponseEditor) SetStatusText(text string) *ResponseEditor {
	e.resp.StatusText = text
	return e
}

// SetVersion changes the HTTP version
func (e *ResponseEditor) SetVersion(version string) *ResponseEditor {
	e.resp.Version = version
	return e
}

// AddHeader adds a new header (preserves order)
func (e *ResponseEditor) AddHeader(name, value string) *ResponseEditor {
	e.resp.Headers.Set(name, value)
	return e
}

// RemoveHeader removes a header
func (e *ResponseEditor) RemoveHeader(name string) *ResponseEditor {
	e.resp.Headers.Del(name)
	return e
}

// UpdateHeader updates an existing header value
func (e *ResponseEditor) UpdateHeader(name, value string) *ResponseEditor {
	e.resp.Headers.Set(name, value)
	return e
}

// SetBody sets the response body (automatically handles compression if needed)
func (e *ResponseEditor) SetBody(body []byte, compress bool) *ResponseEditor {
	e.resp.SetBody(body, compress)
	return e
}

// SetBodyString sets the response body from string
func (e *ResponseEditor) SetBodyString(body string, compress bool) *ResponseEditor {
	e.resp.SetBody([]byte(body), compress)
	return e
}

// RemoveCompression removes compression and decompresses the body
func (e *ResponseEditor) RemoveCompression() *ResponseEditor {
	if e.resp.Compressed {
		e.resp.RawBody = e.resp.Body
		e.resp.Compressed = false
		e.resp.Headers.Del("Content-Encoding")
		e.resp.UpdateContentLength()
	}
	return e
}
