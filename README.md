# HTTPTools

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/WhileEndless/go-httptools)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)

A comprehensive HTTP toolkit for Go, featuring fault-tolerant parsing, raw socket communication, and message editing capabilities. Build HTTP proxies, security testing tools, and network utilities with ease.

## Features

### HTTP Parsing & Editing
- **🔧 Fault-tolerant parsing** of raw HTTP requests and responses
- **📋 Header order preservation** for exact reconstruction
- **🎯 Non-standard header support** (`test:deneme`, malformed headers)
- **🗜️ Automatic decompression** (gzip, deflate, brotli)
- **✏️ Parse → Edit → Rebuild** pipeline
- **📐 Exact format preservation** (spacing, line endings, formatting)

### Raw Socket Communication (NEW)
- **🚀 Raw TCP socket** - Send/receive raw HTTP over TCP/TLS
- **🔒 Full TLS support** - HTTPS with custom certificates and skip verify
- **🔄 Connection pooling** - Keep-Alive and connection reuse
- **⏱️ Detailed timing** - DNS, TCP, TLS, TTFB metrics
- **🔀 Proxy support** - Chain through upstream HTTP/SOCKS proxies
- **🎯 Protocol negotiation** - HTTP/1.1, HTTP/2, H2C support
- **📊 Connection metadata** - Track actual IPs, ports, protocols
- **⚡ Minimal dependencies** - Only standard library + http2

## Quick Start

### Standard Parsing (with normalization)

```go
package main

import (
    "fmt"
    "github.com/WhileEndless/go-httptools/pkg/request"
)

func main() {
    rawReq := []byte(`GET /api/users HTTP/1.1
Host: example.com
test:deneme
Authorization: Bearer token123

`)

    req, err := request.Parse(rawReq)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Method: %s\n", req.Method)
    fmt.Printf("Custom header: %s\n", req.Headers.Get("test"))
    
    // Edit and rebuild
    req.Headers.Set("Authorization", "Bearer new-token")
    rebuilt := req.Build()
    fmt.Println(string(rebuilt))
}
```

### Raw Format Preservation (exact formatting)

```go
package main

import (
    "fmt"
    "github.com/WhileEndless/go-httptools/pkg/request"
)

func main() {
    // Malformed request with weird spacing
    original := []byte(`POST    /api/test   HTTP/1.1
Host:   example.com  
test:deneme
Content-Type:  application/json  

{  "data":  "test"  }`)

    // Parse preserving exact format
    rawReq, err := request.ParseRaw(original)
    if err != nil {
        panic(err)
    }
    
    // Access parsed data normally
    fmt.Printf("Method: %s\n", rawReq.Method)
    fmt.Printf("Custom header: %s\n", rawReq.Headers.Get("test"))
    
    // Edit one header
    rawReq.Headers.Set("Authorization", "Bearer token")
    
    // Rebuild with exact formatting preserved
    rebuilt := rawReq.BuildRaw()
    
    // Original spacing, line endings, formatting preserved!
    fmt.Printf("Identical: %t\n", string(original) == string(rebuilt))
}
```

## Key Capabilities

### 1. **Fault Tolerance**
Handles malformed HTTP messages gracefully:
```go
malformed := []byte(`GET /path
Host: example.com
: empty-header-name
Invalid-Header-No-Colon
test:deneme

`)

req, _ := request.ParseRaw(malformed) // No error!
fmt.Println(req.Headers.Get("test")) // "deneme"
```

### 2. **Header Order Preservation & Positioning**
Maintains exact header ordering with precise control:
```go
rawReq := []byte(`POST / HTTP/1.1
Host: example.com
User-Agent: Mozilla/5.0
test:deneme
Content-Type: application/json

`)

req, _ := request.ParseRaw(rawReq)

// Add header at specific positions
req.Headers.SetAfter("Authorization", "Bearer token", "Host")     // After Host
req.Headers.SetBefore("Cookie", "session=123", "User-Agent")     // Before User-Agent  
req.Headers.SetAt("X-Custom", "value", 0)                        // At index 0 (first)

// Result order: X-Custom, Host, Authorization, Cookie, User-Agent, test, Content-Type
// req.Headers.Get("test") returns "deneme" - custom headers preserved!
```

### 3. **Non-Standard Header Support**
Works with any header format:
```go
// All these work perfectly:
test:deneme                    // No space after colon
Weird Header Name: value       // Spaces in name  
X-123-Numbers: value          // Numbers in name
header-without-value:         // Empty value
```

### 4. **Exact Format Preservation**
Preserves spacing, line endings, formatting:
```go
// Weird spacing preserved exactly
original := `GET     /path     HTTP/1.1
Host:example.com
test:deneme

`
rawReq, _ := request.ParseRaw([]byte(original))
rebuilt := rawReq.BuildRaw()
// string(original) == string(rebuilt) ✅
```

### 5. **Flexible Line Endings**
Supports both `\r\n` and `\n` automatically:
```go
// Works with Unix line endings (\n)
unix := "GET / HTTP/1.1\nHost: example.com\ntest:deneme\n\n"

// Works with Windows line endings (\r\n)  
windows := "GET / HTTP/1.1\r\nHost: example.com\r\ntest:deneme\r\n\r\n"

// Both parse correctly and preserve original format
```

## Burp Suite-like Editing

```go
editor := utils.NewRequestEditor(req)
modified := editor.
    SetMethod("PUT").
    SetURL("/api/users/123").
    AddHeader("Authorization", "Bearer token").
    UpdateHeader("Content-Type", "application/json").
    SetBodyString(`{"updated": true}`).
    AddQueryParam("force", "true").
    GetRequest()

rebuilt := modified.Build()
```

## Raw Socket Communication

**NEW: `rawhttp` package** - Send raw HTTP requests over TCP/TLS sockets with full control.

### Quick Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/WhileEndless/go-httptools/pkg/rawhttp"
)

func main() {
    sender := rawhttp.NewSender()
    defer sender.Close()

    rawRequest := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")

    opts := rawhttp.Options{
        Scheme: "https",
        Host:   "example.com",
        Port:   443,
        ReuseConnection: true, // Enable connection pooling
        InsecureSkipVerify: false,
    }

    resp, err := sender.Do(context.Background(), rawRequest, opts)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Status: %d\n", resp.StatusCode)
    fmt.Printf("Raw Response:\n%s\n", resp.Raw)
    fmt.Printf("Connected to: %s:%d\n", resp.ConnectedIP, resp.ConnectedPort)
    fmt.Printf("Protocol: %s\n", resp.Protocol)
    fmt.Printf("Timing: %s\n", resp.Timing)
}
```

### Key Features

#### 1. **Raw Response Preservation**
The complete response (status + headers + body) is preserved exactly as received from the TCP socket:

```go
resp, _ := sender.Do(ctx, rawRequest, opts)
fmt.Println(string(resp.Raw)) // Exact bytes from TCP socket
```

#### 2. **Connection Pooling**
Automatic Keep-Alive and connection reuse for better performance:

```go
opts := rawhttp.Options{
    Scheme: "https",
    Host:   "example.com",
    ReuseConnection: true, // Enable pooling
}

// Multiple requests reuse the same TCP connection
for i := 0; i < 10; i++ {
    resp, _ := sender.Do(ctx, rawRequest, opts)
}
```

#### 3. **Upstream Proxy Support**
Chain through HTTP or SOCKS proxies:

```go
opts := rawhttp.Options{
    Scheme:   "https",
    Host:     "example.com",
    ProxyURL: "http://proxy.example.com:8080",
}
```

#### 4. **TLS Configuration**
Full TLS control with custom certificates and verification options:

```go
opts := rawhttp.Options{
    Scheme:             "https",
    Host:               "self-signed.example.com",
    InsecureSkipVerify: true, // Skip certificate verification
    CustomCACerts:      [][]byte{pemCert}, // Custom CA certs
    DisableSNI:         false,
}
```

#### 5. **Connection Metadata**
Track actual connection details:

```go
resp, _ := sender.Do(ctx, rawRequest, opts)
fmt.Printf("Connected IP: %s\n", resp.ConnectedIP)
fmt.Printf("Connected Port: %d\n", resp.ConnectedPort)
fmt.Printf("Protocol: %s\n", resp.Protocol) // "HTTP/1.1" or "HTTP/2"
```

#### 6. **Detailed Timing Metrics**
Measure every phase of the request:

```go
resp, _ := sender.Do(ctx, rawRequest, opts)
fmt.Printf("DNS Lookup: %v\n", resp.Timing.DNSLookup)
fmt.Printf("TCP Connect: %v\n", resp.Timing.TCPConnect)
fmt.Printf("TLS Handshake: %v\n", resp.Timing.TLSHandshake)
fmt.Printf("Time to First Byte: %v\n", resp.Timing.TTFB)
fmt.Printf("Total: %v\n", resp.Timing.Total)
```

#### 7. **Protocol Negotiation**
HTTP/1.1, HTTP/2, and H2C support:

```go
opts := rawhttp.Options{
    ForceHTTP1: false, // Allow HTTP/2 via ALPN
    ForceHTTP2: false, // Don't force HTTP/2
    EnableH2C:  false, // HTTP/2 cleartext for non-TLS
}
```

#### 8. **Error Categorization**
Detailed error types for better handling:

```go
resp, err := sender.Do(ctx, rawRequest, opts)
if err != nil {
    if httpErr, ok := err.(*rawhttp.HTTPError); ok {
        switch httpErr.Type {
        case rawhttp.ErrorTypeDNS:
            fmt.Println("DNS resolution failed")
        case rawhttp.ErrorTypeConnection:
            fmt.Println("Connection failed")
        case rawhttp.ErrorTypeTLS:
            fmt.Println("TLS handshake failed")
        case rawhttp.ErrorTypeTimeout:
            fmt.Println("Request timeout")
        }
    }
}
```

### Configuration Options

```go
type Options struct {
    // Connection
    Scheme string // "http" or "https"
    Host   string
    Port   int    // Default: 80 (HTTP), 443 (HTTPS)
    ConnIP string // Optional: specific IP to connect to

    // Timeouts
    ConnTimeout  time.Duration // Default: 30s
    ReadTimeout  time.Duration // Default: 30s
    WriteTimeout time.Duration // Default: 30s

    // TLS
    DisableSNI         bool
    InsecureSkipVerify bool
    CustomCACerts      [][]byte // PEM format

    // Performance
    BodyMemLimit    int64 // Default: 4MB
    ReuseConnection bool  // Default: true

    // Proxy
    ProxyURL string // e.g., "http://proxy:8080"

    // Protocol
    ForceHTTP1 bool
    ForceHTTP2 bool
    EnableH2C  bool // HTTP/2 cleartext
}
```

## API Overview

### RawHTTP Package (NEW)
- `rawhttp.NewSender()` - Create a new sender with connection pooling
- `sender.Do(ctx, rawRequest, opts)` - Send raw HTTP request
- `sender.Close()` - Close all pooled connections
- **Response fields**: `Raw`, `StatusCode`, `Headers`, `Body`, `ConnectedIP`, `Protocol`, `Timing`
- **Error types**: `ErrorTypeDNS`, `ErrorTypeConnection`, `ErrorTypeTLS`, `ErrorTypeTimeout`

### Request Package
- `request.Parse([]byte)` - Standard parsing with normalization
- `request.ParseRaw([]byte)` - Exact format preservation parsing
- `req.Build()` - Standard rebuild
- `rawReq.BuildRaw()` - Exact format rebuild

### Response Package
- `response.Parse([]byte)` - Parse with automatic decompression
- `resp.Build()` - Rebuild (compressed if original was compressed)
- `resp.BuildDecompressed()` - Rebuild with decompressed body

### Headers Package
- `headers.OrderedHeaders` - Standard headers with order preservation
- `headers.OrderedHeadersRaw` - Raw headers with exact format preservation
- **Positioning methods**: `SetAfter()`, `SetBefore()`, `SetAt()`
- Case-insensitive lookups, original case preservation

### Utils Package
- `RequestEditor` / `ResponseEditor` - Burp Suite-like editing
- `ValidateRequest()` / `ValidateResponse()` - Validation with warnings
- Standard library conversion utilities

## Examples

See `examples/` directory for:
- `basic_parsing.go` - Basic parsing examples
- `editing_requests.go` - Request editing workflows
- `editing_responses.go` - Response editing with compression
- `fault_tolerance.go` - Malformed request handling
- `exact_preservation.go` - Format preservation demo
- `header_positioning.go` - Header positioning examples
- `burp_like_usage.go` - Burp Suite-like header management
- `test_rawhttp_features.go` - **NEW** Comprehensive rawhttp feature demonstration

## Testing

```bash
# Run all tests
go test ./tests/...

# Run specific test suites
go test ./tests/unit/           # Unit tests
go test ./tests/integration/    # Integration tests

# Test examples
go run examples/exact_preservation.go
```

## Installation

```bash
go get github.com/WhileEndless/go-httptools
```

## Version Information

```go
package main

import (
    "fmt"
    "github.com/WhileEndless/go-httptools/pkg/version"
)

func main() {
    fmt.Println("HTTPTools version:", version.GetVersion()) // Output: 1.0.0
}
```

Current version: **1.0.0**

## Use Cases

- **Security Testing Tools** - Parse and modify HTTP requests/responses
- **HTTP Proxies** - Intercept and edit traffic while preserving format  
- **Web Scrapers** - Handle malformed HTTP responses gracefully
- **API Testing** - Edit requests with exact control over formatting
- **Protocol Research** - Analyze and reconstruct HTTP messages precisely
- **Burp Suite Extensions** - Go-based tools with similar capabilities

## Dependencies

- **Go 1.21+**
- **github.com/andybalholm/brotli** - Brotli compression support
- **golang.org/x/net/http2** - HTTP/2 protocol support

Minimal external dependencies - core functionality uses only Go standard library.

