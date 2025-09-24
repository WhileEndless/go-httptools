# HTTPTools Architecture

**Version: 1.0.0**

## Design Principles

1. **Fault Tolerance**: Parse malformed HTTP messages gracefully
2. **Header Order Preservation**: Maintain exact header ordering for reconstruction
3. **Non-Standard Support**: Handle non-standard headers and formats
4. **Automatic Compression**: Transparent compression/decompression
5. **Zero Dependencies**: Only external dependency is Brotli compression
6. **Burp Suite Compatibility**: Similar editing workflow and capabilities

## Package Structure

```
httptools/
├── pkg/
│   ├── request/           # HTTP request parsing and building
│   │   ├── request.go     # Core request type and methods
│   │   ├── parser.go      # Fault-tolerant parsing logic
│   │   └── builder.go     # Request reconstruction
│   │
│   ├── response/          # HTTP response parsing and building
│   │   ├── response.go    # Core response type and methods
│   │   ├── parser.go      # Parsing with auto-decompression
│   │   └── builder.go     # Response reconstruction
│   │
│   ├── headers/           # Header management with order preservation
│   │   ├── ordered.go     # OrderedHeaders implementation
│   │   └── parser.go      # Header parsing logic
│   │
│   ├── compression/       # Compression algorithms
│   │   └── compression.go # Gzip, Deflate, Brotli support
│   │
│   ├── errors/            # Structured error types
│   │   └── errors.go      # Error definitions and utilities
│   │
│   └── utils/             # Utility functions and editors
│       ├── editor.go      # Request/Response editors
│       ├── converter.go   # Standard library conversion
│       └── validator.go   # Validation utilities
│
├── examples/              # Usage examples
├── tests/                 # Test suite
└── docs/                  # Documentation
```

## Core Components

### Header Management System

The library provides two complementary header management systems:

#### OrderedHeaders (Standard)
The `OrderedHeaders` type provides normalized header management:

- **Order Preservation**: Headers maintain insertion order
- **Case-Insensitive Lookup**: Standard HTTP header behavior
- **Thread Safety**: Safe for concurrent access
- **Original Case Storage**: Preserves original header name casing
- **Positioning Control**: SetAfter, SetBefore, SetAt methods

```go
type OrderedHeaders struct {
    mu     sync.RWMutex
    order  []string          // Insertion order (lowercase)
    values map[string]string // Case-insensitive storage
    raw    map[string]string // Original case preservation
}
```

#### OrderedHeadersRaw (Exact Format Preservation)
The `OrderedHeadersRaw` type provides pixel-perfect format preservation:

- **Exact Formatting**: Preserves spacing, line endings, case
- **Order Preservation**: Maintains exact insertion order
- **Thread Safety**: Safe for concurrent access
- **Original Line Storage**: Stores complete original header lines
- **Positioning Control**: Same positioning methods as OrderedHeaders

```go
type OrderedHeadersRaw struct {
    mu        sync.RWMutex
    headers   []RawHeader       // Preserves exact order and formatting
    lookup    map[string]int    // Case-insensitive name -> last index
    rawLookup map[string]string // Case-insensitive name -> original case
}

type RawHeader struct {
    Name         string // Parsed name (for lookups)
    Value        string // Parsed value (for lookups)
    OriginalLine string // Exact original line including spacing
}
```

### Request/Response Types

The library provides dual-mode request/response handling:

#### Standard Types (Request/Response)
- **Normalized Processing**: Clean, standardized formatting
- **Raw Data Storage**: Original bytes preserved for reference
- **Parsed Components**: Structured access to HTTP components
- **Clone Support**: Deep copying for safe concurrent use
- **OrderedHeaders**: Uses standard header management

#### Raw Types (RawRequest/RawResponse)  
- **Format Preservation**: Maintains exact original formatting
- **Section Storage**: Separate storage for request line, headers, body
- **Perfect Reconstruction**: Byte-for-byte identical rebuilds
- **OrderedHeadersRaw**: Uses exact format preservation
- **Conversion Support**: Convert to/from standard types

```go
// Standard Request
type Request struct {
    Method  string                 
    URL     string                 
    Version string                 
    Headers *headers.OrderedHeaders
    Body    []byte                 
    Raw     []byte                 
}

// Raw Request (Format Preservation)
type RawRequest struct {
    Method        string                     
    URL           string                     
    Version       string                     
    Headers       *headers.OrderedHeadersRaw
    Body          []byte                     
    Raw           []byte                     
    RequestLine   string                     // Exact original request line
    HeaderSection []byte                     // Exact original header section
    BodySection   []byte                     // Exact original body section
}
```

### Fault Tolerance Strategy

The library implements multiple levels of fault tolerance:

1. **Parser Level**: Continue parsing despite errors
2. **Validation Level**: Identify issues without blocking usage
3. **Reconstruction Level**: Build valid HTTP messages from parsed data

## Parsing Flow

### Standard Request Parsing

```
Raw Bytes → Split Lines → Parse Request Line → Parse Headers → Parse Body
     ↓           ↓              ↓               ↓            ↓
Store Raw → Scanner → Method/URL/Version → OrderedHeaders → Body Bytes
```

### Raw Request Parsing (Format Preservation)

```
Raw Bytes → Section Split → Parse Request Line → Parse Headers Raw → Store Sections
     ↓           ↓              ↓                    ↓               ↓
Store Raw → Line/Headers/Body → Method/URL/Version → OrderedHeadersRaw → Exact Storage
```

### Response Parsing

```
Raw Bytes → Split Lines → Parse Status Line → Parse Headers → Parse Body → Decompress
     ↓           ↓              ↓               ↓            ↓         ↓
Store Raw → Scanner → Version/Code/Text → OrderedHeaders → Body Bytes → Decompressed
```

### Header Positioning Architecture

```
Header Operation → Find Reference → Calculate Position → Insert/Update → Rebuild Lookup
       ↓               ↓                ↓                  ↓             ↓
   SetAfter()     Locate "Host"    Position + 1      Insert at Index    Update Maps
   SetBefore()    Locate "User-Agent"  Position      Insert at Index    Update Maps  
   SetAt()        Validate Index       Index         Insert at Index    Update Maps
```

## Compression Architecture

The compression system supports three algorithms:

- **Gzip**: Standard web compression
- **Deflate**: Raw deflate compression  
- **Brotli**: Google's high-efficiency compression

```go
type CompressionType int

const (
    CompressionNone
    CompressionGzip
    CompressionDeflate
    CompressionBrotli
)
```

Responses store both compressed (`RawBody`) and decompressed (`Body`) versions for flexibility.

## Editor Pattern

The editor classes provide a fluent interface for HTTP message modification:

```go
editor := utils.NewRequestEditor(req)
modified := editor.
    SetMethod("PUT").
    AddHeader("Auth", "Bearer token").
    SetBodyString(`{"data": "value"}`).
    GetRequest()
```

This pattern allows:
- **Chaining**: Multiple operations in sequence
- **Immutability**: Original request/response unchanged
- **Validation**: Built-in validation at each step

## Error Handling Strategy

Structured errors provide detailed context:

```go
type Error struct {
    Type    ErrorType  // Categorized error type
    Message string     // Human-readable message
    Context string     // Where the error occurred
    Raw     []byte     // Original data causing error
}
```

Error types allow specific handling:
- `ErrorTypeInvalidFormat`: Malformed HTTP structure
- `ErrorTypeMalformedHeader`: Bad header format
- `ErrorTypeCompressionError`: Compression/decompression issues

## Thread Safety

| Component | Thread Safety | Notes |
|-----------|--------------|-------|
| OrderedHeaders | ✅ Safe | Uses RWMutex for protection |
| Request | ❌ Not Safe | Clone for concurrent use |
| Response | ❌ Not Safe | Clone for concurrent use |
| Editors | ❌ Not Safe | Create separate instances |
| Parsers | ✅ Safe | Stateless functions |

## Memory Management

The library is designed for memory efficiency:

- **Lazy Compression**: Only decompress when accessed
- **Selective Copying**: Clone only copies necessary data
- **Buffer Reuse**: Internal buffers reused where possible
- **Raw Preservation**: Original data kept for perfect reconstruction

## Validation Architecture

Three-tier validation system:

1. **Parse-time**: Minimal validation, maximum tolerance
2. **Explicit Validation**: Comprehensive checks via `ValidateRequest/Response`
3. **Build-time**: Final validation before reconstruction

Validation results categorized as:
- **Errors**: Fatal issues preventing proper operation
- **Warnings**: Non-fatal issues that may indicate problems

## Extension Points

The architecture supports extension through:

- **Custom Compression**: Add new compression algorithms
- **Custom Validators**: Implement domain-specific validation
- **Custom Editors**: Build specialized editing workflows
- **Custom Converters**: Support additional HTTP representations

## Performance Considerations

- **Single-pass Parsing**: Minimize memory allocations
- **String Interning**: Reuse common header names
- **Buffer Pooling**: Reuse buffers for large operations
- **Compression Caching**: Cache decompression results

## Integration Patterns

Common integration approaches:

1. **Proxy Integration**: Parse → Inspect → Modify → Forward
2. **Testing Tools**: Parse → Validate → Generate Test Cases
3. **Security Analysis**: Parse → Extract Patterns → Analyze
4. **Protocol Bridge**: Parse HTTP/1.1 → Convert → HTTP/2