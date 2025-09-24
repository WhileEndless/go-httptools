# Header Positioning Guide

**Version: 1.0.0**

HTTPTools provides precise control over HTTP header positioning, similar to Burp Suite's capabilities.

## Overview

With HTTPTools, you can:
- Add headers at specific positions
- Insert headers relative to existing headers
- Maintain exact formatting while editing
- Preserve custom headers like `test:deneme`

## Methods Available

### 1. SetAfter(name, value, afterHeader)
Adds a header immediately after the specified header.

```go
// Add Authorization after Host header
rawReq.Headers.SetAfter("Authorization", "Bearer token123", "Host")

// Result:
// Host: example.com
// Authorization: Bearer token123  ← Added here
// User-Agent: Mozilla/5.0
```

### 2. SetBefore(name, value, beforeHeader)
Adds a header immediately before the specified header.

```go
// Add Cookie before User-Agent header
rawReq.Headers.SetBefore("Cookie", "session=abc123", "User-Agent")

// Result:
// Host: example.com
// Cookie: session=abc123  ← Added here
// User-Agent: Mozilla/5.0
```

### 3. SetAt(name, value, index)
Adds a header at a specific index position (0-based).

```go
// Add X-Custom at index 0 (first position)
rawReq.Headers.SetAt("X-Custom", "value", 0)

// Result:
// X-Custom: value  ← Added at index 0
// Host: example.com
// User-Agent: Mozilla/5.0
```

### 4. Set(name, value)
Standard method - adds header at the end.

```go
// Add API-Key at the end
rawReq.Headers.Set("API-Key", "secret123")

// Result:
// Host: example.com
// User-Agent: Mozilla/5.0
// API-Key: secret123  ← Added at end
```

## Practical Examples

### Example 1: Authentication Headers
```go
original := []byte(`POST /api/login HTTP/1.1
Host: api.example.com
User-Agent: Mozilla/5.0
test:deneme
Content-Type: application/json

{"user":"admin"}`)

rawReq, _ := request.ParseRaw(original)

// Add auth headers in specific positions
rawReq.Headers.SetAfter("Authorization", "Bearer eyJ0eXAi...", "Host")
rawReq.Headers.SetAfter("X-API-Key", "secret123", "Authorization")

// Result order: Host → Authorization → X-API-Key → User-Agent → test → Content-Type
```

### Example 2: Proxy Headers
```go
// Add proxy-related headers
rawReq.Headers.SetBefore("X-Forwarded-For", "127.0.0.1", "User-Agent")
rawReq.Headers.SetBefore("X-Real-IP", "192.168.1.100", "User-Agent")

// Both headers inserted before User-Agent in order
```

### Example 3: Security Headers
```go
// Add security headers at the beginning
rawReq.Headers.SetAt("X-Requested-With", "XMLHttpRequest", 0)
rawReq.Headers.SetAfter("X-CSRF-Token", "abc123def456", "X-Requested-With")

// Security headers first, then original headers
```

## Behavior Details

### Existing Header Updates
If a header already exists, positioning methods update it in place (no position change):

```go
// Authorization already exists at position 2
rawReq.Headers.SetAfter("Authorization", "Bearer new-token", "Host")

// Authorization stays at position 2 with new value
```

### Fallback Behavior
If the reference header is not found, the new header is added at the end:

```go
// If "NonExistent" header doesn't exist
rawReq.Headers.SetAfter("New-Header", "value", "NonExistent")

// New-Header is added at the end (same as Set())
```

### Case Sensitivity
Header names are case-insensitive for lookups but preserve original case:

```go
rawReq.Headers.SetAfter("auth", "token", "HOST")  // Works fine
// Finds "Host" header regardless of case
// Preserves "auth" case as provided
```

## Format Preservation

### Raw Format Preservation
When using `ParseRaw()` and `BuildRaw()`, exact formatting is preserved:

```go
original := []byte(`POST   /api   HTTP/1.1
Host:   example.com  
test:deneme
User-Agent:Mozilla/5.0    

`)

rawReq, _ := request.ParseRaw(original)
rawReq.Headers.SetAfter("Authorization", "Bearer token", "Host")

rebuilt := rawReq.BuildRaw()
// Original spacing and formatting preserved exactly!
```

### Standard Format
When using `Parse()` and `Build()`, formatting is normalized:

```go
req, _ := request.Parse(original)
req.Headers.SetAfter("Authorization", "Bearer token", "Host")

rebuilt := req.Build()
// Clean, normalized formatting with proper CRLF and spacing
```

## Integration with Burp Suite Workflow

HTTPTools positioning methods enable Burp Suite-like header manipulation:

```go
// 1. Parse intercepted request
intercepted := []byte(`POST /login HTTP/1.1
Host: target.com
User-Agent: Mozilla/5.0
test:deneme
Content-Type: application/json

{"user":"test"}`)

rawReq, _ := request.ParseRaw(intercepted)

// 2. Add authentication (like in Burp's Repeater)
rawReq.Headers.SetAfter("Authorization", "Bearer token123", "Host")

// 3. Add session cookie (before User-Agent)
rawReq.Headers.SetBefore("Cookie", "PHPSESSID=abc123", "User-Agent")

// 4. Add API key (after Content-Type)
rawReq.Headers.SetAfter("X-API-Key", "secret", "Content-Type")

// 5. Send modified request
modified := rawReq.BuildRaw()
// Send 'modified' to target
```

## Performance Notes

- Header positioning operations are O(n) where n is the number of headers
- For bulk operations, consider batching changes
- Raw format preservation has minimal performance overhead
- Thread-safe for concurrent reads, but not writes

## Error Handling

Positioning methods are designed to be fault-tolerant:
- Invalid indices are clamped to valid range
- Missing reference headers cause fallback to end insertion
- No exceptions thrown - operations always succeed

This ensures robust operation even with malformed or unexpected HTTP messages.

## Best Practices

1. **Use Raw parsing for exact format preservation**
2. **Use positioning methods for precise control**
3. **Custom headers like `test:deneme` are fully supported**
4. **Chain operations for complex modifications**
5. **Test with your specific HTTP message formats**

```go
// Good practice: Chain operations
rawReq.Headers.
    SetAfter("Authorization", "Bearer token", "Host").
    SetBefore("Cookie", "session=123", "User-Agent").
    SetAt("X-Priority", "high", 0)
```