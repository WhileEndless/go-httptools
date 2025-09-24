# HTTPTools Examples

**Version: 1.0.0**

## Basic Request Parsing

### Standard Parsing (Normalized)
```go
rawRequest := []byte(`GET /api/users?page=1 HTTP/1.1
Host: example.com
User-Agent: Mozilla/5.0
test:deneme
Authorization: Bearer token123

`)

req, err := request.Parse(rawRequest)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Method: %s\n", req.Method)
fmt.Printf("URL: %s\n", req.URL)
fmt.Printf("Custom header: %s\n", req.Headers.Get("test"))

// Rebuild with normalized formatting
rebuilt := req.Build()
```

### Raw Parsing (Exact Format Preservation)
```go
// Request with weird spacing - will be preserved exactly
rawRequest := []byte(`GET    /api/users?page=1   HTTP/1.1
Host:   example.com  
User-Agent:Mozilla/5.0    
test:deneme
Authorization:   Bearer   token123  

`)

rawReq, err := request.ParseRaw(rawRequest)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Method: %s\n", rawReq.Method)
fmt.Printf("Custom header: %s\n", rawReq.Headers.Get("test"))

// Rebuild with EXACT original formatting preserved
rebuilt := rawReq.BuildRaw()
// string(rawRequest) == string(rebuilt) ✅ Identical!
```

## Basic Response Parsing with Compression

```go
// Response with gzip compression
rawResponse := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Content-Encoding: gzip
Content-Length: 123

` + gzippedContent)

resp, err := response.Parse(rawResponse)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Compressed: %t\n", resp.Compressed)
fmt.Printf("Decompressed body: %s\n", string(resp.Body))
```

## Request Editing (Burp Suite Style)

```go
// Parse original request
req, _ := request.Parse(rawRequest)

// Chain edits
editor := utils.NewRequestEditor(req)
modifiedReq := editor.
    SetMethod("PUT").
    SetURL("/api/users/123").
    AddHeader("Authorization", "Bearer new-token").
    UpdateHeader("Content-Type", "application/json").
    SetBodyString(`{"name":"updated"}`).
    AddQueryParam("force", "true").
    GetRequest()

// Rebuild request
newRaw := modifiedReq.Build()
```

## Response Editing

```go
// Parse and edit response
resp, _ := response.Parse(rawResponse)

editor := utils.NewResponseEditor(resp)
modifiedResp := editor.
    SetStatusCode(200).
    SetStatusText("OK").
    UpdateHeader("Content-Type", "application/json").
    SetBodyString(`{"success":true}`, false). // Don't compress
    GetResponse()

fmt.Println(modifiedResp.BuildString())
```

## Header Order Preservation & Positioning

### Basic Order Preservation
```go
rawReq := []byte(`POST /api HTTP/1.1
Host: example.com
X-Custom: first
Content-Type: application/json
X-Custom: second
test:deneme

`)

req, _ := request.Parse(rawReq)

// Headers maintain insertion order
for i, header := range req.Headers.All() {
    fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
}
// Output:
// 1. Host: example.com
// 2. X-Custom: first
// 3. Content-Type: application/json
// 4. X-Custom: second
// 5. test: deneme

// Get() returns last value for duplicates
fmt.Println(req.Headers.Get("X-Custom")) // "second"
fmt.Println(req.Headers.Get("test"))     // "deneme"
```

### Header Positioning (NEW!)
```go
rawReq := []byte(`POST /api HTTP/1.1
Host: example.com
User-Agent: Mozilla/5.0
test:deneme
Content-Type: application/json

`)

rawReq, _ := request.ParseRaw(rawReq)

// Add headers at specific positions
rawReq.Headers.SetAfter("Authorization", "Bearer token", "Host")      // After Host
rawReq.Headers.SetBefore("Cookie", "session=123", "User-Agent")      // Before User-Agent  
rawReq.Headers.SetAt("X-API-Key", "secret", 0)                       // At index 0 (first)
rawReq.Headers.Set("X-Last", "value")                                 // At end (normal)

// Final order: X-API-Key, Host, Authorization, Cookie, User-Agent, test, Content-Type, X-Last
for i, header := range rawReq.Headers.All() {
    fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
}

// Custom header still accessible
fmt.Println(rawReq.Headers.Get("test")) // "deneme" - preserved!
```

## Fault Tolerance

```go
// Malformed request with various issues
malformed := []byte(`GET /path HTTP/1.1
Host: example.com
: empty-header-name
Invalid-Header-No-Colon
test:deneme

`)

req, err := request.Parse(malformed)
// No error - parsed with fault tolerance

// Check issues with validation
validation := utils.ValidateRequest(req)
fmt.Printf("Valid: %t\n", validation.Valid)
for _, warning := range validation.Warnings {
    fmt.Printf("Warning: %s\n", warning)
}
```

## Compression Handling

```go
// Automatic decompression
resp, _ := response.Parse(gzipResponse)
fmt.Printf("Original (compressed): %d bytes\n", len(resp.RawBody))
fmt.Printf("Decompressed: %d bytes\n", len(resp.Body))
fmt.Printf("Content: %s\n", string(resp.Body))

// Build with compression preserved
compressed := resp.Build() // Uses RawBody (compressed)

// Build decompressed version
decompressed := resp.BuildDecompressed() // Uses Body (decompressed)
```

## Working with Non-Standard Headers

```go
rawReq := []byte(`GET / HTTP/1.1
Host: example.com
test:deneme
X-Custom-123: value
Weird Header Name: spaces everywhere
Content-Type: application/json

`)

req, _ := request.Parse(rawReq)

// All headers preserved exactly as they were
fmt.Println("Non-standard headers supported:")
fmt.Printf("test: %s\n", req.Headers.Get("test"))
fmt.Printf("Weird Header Name: %s\n", req.Headers.Get("Weird Header Name"))

// Rebuild maintains exact format
rebuilt := req.BuildString()
// Identical to original (with header order preserved)
```

## Validation and Error Checking

```go
// Validate request
validation := utils.ValidateRequest(req)

if !validation.Valid {
    fmt.Println("Request has errors:")
    for _, err := range validation.Errors {
        fmt.Printf("- %s\n", err)
    }
}

if len(validation.Warnings) > 0 {
    fmt.Println("Request warnings:")
    for _, warning := range validation.Warnings {
        fmt.Printf("- %s\n", warning)
    }
}
```

## Converting to/from Standard Library

```go
// Convert to standard http.Request
httpReq, err := utils.ToStandardRequest(req)
if err != nil {
    log.Fatal(err)
}

// Use with standard library
client := &http.Client{}
resp, err := client.Do(httpReq)

// Convert back from standard response
ourResp := utils.FromStandardResponse(resp)
```

## Complex Editing Workflow

```go
// Parse → Validate → Edit → Validate → Build
req, err := request.Parse(rawRequest)
if err != nil {
    log.Fatal(err)
}

// Initial validation
if validation := utils.ValidateRequest(req); !validation.Valid {
    fmt.Println("Original request has issues")
}

// Edit request
editor := utils.NewRequestEditor(req)
fixed := editor.
    SetMethod("POST").
    RemoveHeader("X-Bad-Header").
    AddHeader("Content-Type", "application/json").
    SetBodyString(`{"fixed": true}`).
    GetRequest()

// Final validation
if validation := utils.ValidateRequest(fixed); validation.Valid {
    fmt.Println("Request fixed!")
    finalRaw := fixed.Build()
    // Use finalRaw...
}
```

## Error Handling

```go
req, err := request.Parse(invalidData)
if err != nil {
    if parseErr, ok := err.(*errors.Error); ok {
        switch parseErr.Type {
        case errors.ErrorTypeInvalidFormat:
            fmt.Println("Invalid format:", parseErr.Message)
        case errors.ErrorTypeInvalidMethod:
            fmt.Println("Invalid method:", parseErr.Message)
        }
    }
}
```