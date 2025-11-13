package rawhttp

// Response represents the response from a raw HTTP request
type Response struct {
	// Raw response data (status line + headers + body) exactly as received from TCP socket
	// This is the complete unmodified response including all bytes
	Raw []byte

	// Parsed response fields (optional, can be parsed from Raw if needed)
	StatusCode int
	Headers    map[string][]string
	Body       []byte

	// Connection metadata
	ConnectedIP   string // Actual IP address connected to (after DNS resolution)
	ConnectedPort int    // Actual port connected to
	Protocol      string // Negotiated protocol: "HTTP/1.1" or "HTTP/2"

	// Timing information
	Timing *Timing

	// Error information (if any)
	Error error
}

// NewResponse creates a new Response instance
func NewResponse() *Response {
	return &Response{
		Headers: make(map[string][]string),
		Timing:  &Timing{},
	}
}

// GetHeader returns the first value for a given header name (case-insensitive)
func (r *Response) GetHeader(name string) string {
	values := r.Headers[name]
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetHeaders returns all values for a given header name (case-insensitive)
func (r *Response) GetHeaders(name string) []string {
	return r.Headers[name]
}

// SetHeader sets a header value (replaces existing)
func (r *Response) SetHeader(name, value string) {
	r.Headers[name] = []string{value}
}

// AddHeader adds a header value (appends to existing)
func (r *Response) AddHeader(name, value string) {
	r.Headers[name] = append(r.Headers[name], value)
}

// IsSuccessful returns true if the response has a 2xx status code
func (r *Response) IsSuccessful() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsRedirect returns true if the response has a 3xx status code
func (r *Response) IsRedirect() bool {
	return r.StatusCode >= 300 && r.StatusCode < 400
}

// IsClientError returns true if the response has a 4xx status code
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the response has a 5xx status code
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}
