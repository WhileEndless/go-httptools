# HTTPTools API Documentation

**Version: 1.0.0**

## Overview

HTTPTools provides a comprehensive API for parsing, editing, and rebuilding HTTP requests and responses with fault tolerance and header order preservation.

## Core Types

### Request

```go
type Request struct {
    Method  string                 // HTTP method (GET, POST, etc.)
    URL     string                 // Request URL/path
    Version string                 // HTTP version (HTTP/1.1, HTTP/2, etc.)
    Headers *headers.OrderedHeaders // Headers with preserved order
    Body    []byte                 // Request body
    Raw     []byte                 // Original raw request data
}
```

### Response

```go
type Response struct {
    Version    string                 // HTTP version
    StatusCode int                    // HTTP status code
    StatusText string                 // Status text
    Headers    *headers.OrderedHeaders // Headers with preserved order
    Body       []byte                 // Decompressed response body
    RawBody    []byte                 // Original compressed body (if any)
    Raw        []byte                 // Original raw response data
    Compressed bool                   // Whether original body was compressed
}
```

### OrderedHeaders

```go
type OrderedHeaders struct {
    // Preserves insertion order and handles case-insensitive lookups
}

// Basic Methods
func (h *OrderedHeaders) Set(name, value string)
func (h *OrderedHeaders) Get(name string) string
func (h *OrderedHeaders) GetRaw(name string) string    // Returns original case
func (h *OrderedHeaders) Has(name string) bool
func (h *OrderedHeaders) Del(name string)
func (h *OrderedHeaders) All() []Header
func (h *OrderedHeaders) Len() int

// Positioning Methods (NEW!)
func (h *OrderedHeaders) SetAfter(name, value, afterHeader string)
func (h *OrderedHeaders) SetBefore(name, value, beforeHeader string) 
func (h *OrderedHeaders) SetAt(name, value string, index int)
```

### OrderedHeadersRaw (Exact Format Preservation)

```go
type OrderedHeadersRaw struct {
    // Preserves exact formatting including spacing, case, line endings
}

// Basic Methods
func (h *OrderedHeadersRaw) Set(name, value string)
func (h *OrderedHeadersRaw) Get(name string) string
func (h *OrderedHeadersRaw) GetRaw(name string) string
func (h *OrderedHeadersRaw) Has(name string) bool
func (h *OrderedHeadersRaw) Del(name string)
func (h *OrderedHeadersRaw) All() []RawHeader
func (h *OrderedHeadersRaw) AllStandard() []Header    // Convert to standard format
func (h *OrderedHeadersRaw) Len() int
func (h *OrderedHeadersRaw) BuildRaw() []byte         // Rebuild with exact formatting

// Positioning Methods (NEW!)
func (h *OrderedHeadersRaw) SetAfter(name, value, afterHeader string)
func (h *OrderedHeadersRaw) SetBefore(name, value, beforeHeader string)
func (h *OrderedHeadersRaw) SetAt(name, value string, index int)

// Types
type RawHeader struct {
    Name         string  // Parsed name
    Value        string  // Parsed value  
    OriginalLine string  // Exact original formatting
}
```

## Request Package

### Standard Request Type

```go
type Request struct {
    Method  string                 // HTTP method (GET, POST, etc.)
    URL     string                 // Request URL/path
    Version string                 // HTTP version (HTTP/1.1, HTTP/2, etc.)
    Headers *headers.OrderedHeaders // Headers with preserved order
    Body    []byte                 // Request body
    Raw     []byte                 // Original raw request data
}
```

### Raw Request Type (Exact Format Preservation)

```go
type RawRequest struct {
    Method        string                     // HTTP method (GET, POST, etc.)
    URL           string                     // Request URL/path
    Version       string                     // HTTP version
    Headers       *headers.OrderedHeadersRaw // Headers with exact formatting preserved
    Body          []byte                     // Request body
    Raw           []byte                     // Original raw request data
    RequestLine   string                     // Exact original request line
    HeaderSection []byte                     // Exact original header section
    BodySection   []byte                     // Exact original body section
}
```

### Parsing

```go
func Parse(data []byte) (*Request, error)        // Standard parsing with normalization
func ParseRaw(data []byte) (*RawRequest, error)  // Exact format preservation parsing
```

**Standard Parse** - Handles with fault tolerance:
- Non-standard HTTP methods
- Malformed headers (stores as X-Malformed-Header)
- Missing HTTP version (defaults to HTTP/1.1)
- Empty header names (stores as X-Empty-Header-Name)
- Normalizes formatting (adds proper CRLF, spacing)

**Raw Parse** - Additional features:
- Preserves exact spacing, line endings, formatting
- Supports both `\r\n` and `\n` line endings
- Maintains original header casing and spacing
- Perfect reconstruction capability

### Building

```go
// Standard Request Building
func (r *Request) Build() []byte
func (r *Request) BuildString() string

// Raw Request Building (Exact Format Preservation) 
func (r *RawRequest) BuildRaw() []byte
func (r *RawRequest) BuildRawString() string

// Conversion
func (r *RawRequest) ToStandard() *Request
func FromStandard(req *Request) *RawRequest
```

**Standard Build** - Reconstructs with normalized formatting (CRLF, proper spacing)
**Raw Build** - Reconstructs with exact original formatting preserved

### Utility Methods

```go
func (r *Request) Clone() *Request
func (r *Request) GetContentLength() string
func (r *Request) GetContentType() string
func (r *Request) GetHost() string
func (r *Request) GetUserAgent() string
func (r *Request) SetBody(body []byte)
func (r *Request) IsHTTPS() bool
```

## Response Package

### Parsing

```go
func Parse(data []byte) (*Response, error)
```

Parses raw HTTP response data with automatic decompression. Supports:
- Gzip, Deflate, and Brotli compression
- Invalid status codes (with validation warnings)
- Missing status text (provides defaults)
- Malformed headers (fault tolerance)

### Building

```go
func (r *Response) Build() []byte
func (r *Response) BuildString() string
func (r *Response) BuildDecompressed() []byte
```

Reconstructs the HTTP response:
- `Build()`: Uses original body (compressed if applicable)
- `BuildDecompressed()`: Uses decompressed body, removes Content-Encoding

### Utility Methods

```go
func (r *Response) Clone() *Response
func (r *Response) GetContentLength() int
func (r *Response) GetContentType() string
func (r *Response) GetContentEncoding() string
func (r *Response) GetServer() string
func (r *Response) SetBody(body []byte, compress bool) error
func (r *Response) IsSuccessful() bool
func (r *Response) IsRedirect() bool
func (r *Response) IsClientError() bool
func (r *Response) IsServerError() bool
func (r *Response) GetRedirectLocation() string
```

## Compression Package

### Supported Algorithms

- **Gzip**: Standard gzip compression
- **Deflate**: Deflate compression
- **Brotli**: Google's Brotli compression

### Functions

```go
func DetectCompression(contentEncoding string) CompressionType
func Decompress(data []byte, compressionType CompressionType) ([]byte, error)
func Compress(data []byte, compressionType CompressionType) ([]byte, error)
```

## Utils Package

### RequestEditor - Burp Suite-like Editing

```go
type RequestEditor struct {}

func NewRequestEditor(req *Request) *RequestEditor
func (e *RequestEditor) SetMethod(method string) *RequestEditor
func (e *RequestEditor) SetURL(urlPath string) *RequestEditor
func (e *RequestEditor) SetVersion(version string) *RequestEditor
func (e *RequestEditor) AddHeader(name, value string) *RequestEditor
func (e *RequestEditor) RemoveHeader(name string) *RequestEditor
func (e *RequestEditor) UpdateHeader(name, value string) *RequestEditor
func (e *RequestEditor) SetBody(body []byte) *RequestEditor
func (e *RequestEditor) SetBodyString(body string) *RequestEditor
func (e *RequestEditor) AddQueryParam(key, value string) *RequestEditor
func (e *RequestEditor) SetQueryParam(key, value string) *RequestEditor
func (e *RequestEditor) RemoveQueryParam(key string) *RequestEditor
func (e *RequestEditor) GetRequest() *Request
```

### ResponseEditor

```go
type ResponseEditor struct {}

func NewResponseEditor(resp *Response) *ResponseEditor
func (e *ResponseEditor) SetStatusCode(code int) *ResponseEditor
func (e *ResponseEditor) SetStatusText(text string) *ResponseEditor
func (e *ResponseEditor) SetVersion(version string) *ResponseEditor
func (e *ResponseEditor) AddHeader(name, value string) *ResponseEditor
func (e *ResponseEditor) RemoveHeader(name string) *ResponseEditor
func (e *ResponseEditor) UpdateHeader(name, value string) *ResponseEditor
func (e *ResponseEditor) SetBody(body []byte, compress bool) *ResponseEditor
func (e *ResponseEditor) SetBodyString(body string, compress bool) *ResponseEditor
func (e *ResponseEditor) RemoveCompression() *ResponseEditor
func (e *ResponseEditor) GetResponse() *Response
```

### Validation

```go
type ValidationResult struct {
    Valid    bool
    Warnings []string
    Errors   []string
}

func ValidateRequest(req *Request) *ValidationResult
func ValidateResponse(resp *Response) *ValidationResult
```

Validates HTTP messages and returns detailed warnings/errors for:
- Invalid methods, URLs, versions
- Malformed headers
- Content-Length mismatches
- Compression inconsistencies
- Security issues (newlines in headers)

### Standard Library Conversion

```go
func ToStandardRequest(req *Request) (*http.Request, error)
func FromStandardRequest(httpReq *http.Request) *Request
func ToStandardResponse(resp *Response) *http.Response
func FromStandardResponse(httpResp *http.Response) *Response
```

## Error Handling

All parsing functions return structured errors:

```go
type Error struct {
    Type    ErrorType
    Message string
    Context string
    Raw     []byte
}

// Error types
const (
    ErrorTypeInvalidFormat
    ErrorTypeMalformedHeader
    ErrorTypeInvalidMethod
    ErrorTypeInvalidURL
    ErrorTypeInvalidVersion
    ErrorTypeInvalidStatusCode
    ErrorTypeCompressionError
)
```

## Thread Safety

- **OrderedHeaders**: Thread-safe for concurrent access
- **Request/Response**: Not thread-safe, clone for concurrent use
- **Editors**: Create separate instances for concurrent editing