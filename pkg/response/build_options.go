package response

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/chunked"
	"github.com/WhileEndless/go-httptools/pkg/compression"
)

// CompressionMethod represents compression options for build
type CompressionMethod int

const (
	// CompressionKeep keeps the original compression (default)
	CompressionKeep CompressionMethod = iota
	// CompressionNone removes compression (decompress)
	CompressionNone
	// CompressionGzip compresses with gzip
	CompressionGzip
	// CompressionDeflate compresses with deflate
	CompressionDeflate
	// CompressionBrotli compresses with brotli
	CompressionBrotli
)

// ChunkedOption represents chunked encoding options for build
type ChunkedOption int

const (
	// ChunkedKeep keeps original chunked state (default)
	ChunkedKeep ChunkedOption = iota
	// ChunkedRemove removes chunked encoding (dechunk)
	ChunkedRemove
	// ChunkedApply applies chunked encoding
	ChunkedApply
)

// HTTPVersion represents the HTTP version for build output
type HTTPVersion int

const (
	// HTTPVersionKeep keeps original version (default)
	HTTPVersionKeep HTTPVersion = iota
	// HTTPVersion11 builds as HTTP/1.1
	HTTPVersion11
	// HTTPVersion2 builds as HTTP/2 format
	HTTPVersion2
)

// BuildOptions configures how the response is built
type BuildOptions struct {
	// Compression controls body compression
	// Default: CompressionKeep (preserve original)
	Compression CompressionMethod

	// Chunked controls chunked transfer encoding
	// Default: ChunkedKeep (preserve original)
	Chunked ChunkedOption

	// ChunkSize for chunked encoding (0 = default 8192)
	ChunkSize int

	// HTTPVersion controls output format
	// Default: HTTPVersionKeep
	HTTPVersion HTTPVersion

	// UpdateContentLength automatically updates Content-Length header
	// Default: true
	UpdateContentLength bool

	// UpdateContentEncoding automatically updates Content-Encoding header
	// When decompressing: removes header
	// When compressing: sets appropriate header
	// Default: true
	UpdateContentEncoding bool

	// UpdateTransferEncoding automatically updates Transfer-Encoding header
	// When dechunking: removes "chunked" from header
	// When chunking: adds "chunked" to header
	// Default: true
	UpdateTransferEncoding bool

	// LineSeparator overrides the line separator
	// Empty string means use original
	LineSeparator string

	// PreserveOriginalHeaders keeps original header formatting
	// When false, headers are normalized
	// Default: true
	PreserveOriginalHeaders bool
}

// DefaultBuildOptions returns default build options
// - Keeps original compression and chunked state
// - Updates all headers automatically
// - Preserves original formatting
func DefaultBuildOptions() BuildOptions {
	return BuildOptions{
		Compression:             CompressionKeep,
		Chunked:                 ChunkedKeep,
		ChunkSize:               0,
		HTTPVersion:             HTTPVersionKeep,
		UpdateContentLength:     true,
		UpdateContentEncoding:   true,
		UpdateTransferEncoding:  true,
		LineSeparator:           "",
		PreserveOriginalHeaders: true,
	}
}

// DecompressedOptions returns options for fully decompressed/dechunked output
func DecompressedOptions() BuildOptions {
	opts := DefaultBuildOptions()
	opts.Compression = CompressionNone
	opts.Chunked = ChunkedRemove
	return opts
}

// NormalizedOptions returns options for normalized output
// - Decompressed, dechunked, standard CRLF line endings
func NormalizedOptions() BuildOptions {
	return BuildOptions{
		Compression:             CompressionNone,
		Chunked:                 ChunkedRemove,
		ChunkSize:               0,
		HTTPVersion:             HTTPVersion11,
		UpdateContentLength:     true,
		UpdateContentEncoding:   true,
		UpdateTransferEncoding:  true,
		LineSeparator:           "\r\n",
		PreserveOriginalHeaders: false,
	}
}

// HTTP2Options returns options for HTTP/2 format output
func HTTP2Options() BuildOptions {
	opts := DefaultBuildOptions()
	opts.HTTPVersion = HTTPVersion2
	opts.Chunked = ChunkedRemove // HTTP/2 doesn't use chunked encoding
	return opts
}

// BuildWithOptions builds the response with specified options
func (r *Response) BuildWithOptions(opts BuildOptions) ([]byte, error) {
	// Get line separator
	lineSep := opts.LineSeparator
	if lineSep == "" {
		lineSep = r.LineSeparator
	}
	if lineSep == "" {
		lineSep = "\r\n"
	}

	// Prepare body based on options
	body, err := r.prepareBody(opts)
	if err != nil {
		return nil, err
	}

	// Prepare headers based on options
	headers := r.prepareHeaders(opts, body)

	// Build based on HTTP version
	switch opts.HTTPVersion {
	case HTTPVersion2:
		return r.buildHTTP2Format(headers, body, lineSep), nil
	default:
		return r.buildHTTP1Format(headers, body, lineSep, opts.PreserveOriginalHeaders), nil
	}
}

// prepareBody processes body according to options
func (r *Response) prepareBody(opts BuildOptions) ([]byte, error) {
	// Start with appropriate source body
	var body []byte

	// Step 1: Get the base body (handle chunked first)
	if r.IsBodyChunked && opts.Chunked == ChunkedRemove {
		// Dechunk the body
		if len(r.RawBody) > 0 {
			decoded, _ := chunked.Decode(r.RawBody)
			body = decoded
		} else {
			decoded, _ := chunked.Decode(r.Body)
			body = decoded
		}
	} else if r.IsBodyChunked && opts.Chunked == ChunkedKeep {
		// Keep chunked, use RawBody if available
		if len(r.RawBody) > 0 {
			body = r.RawBody
		} else {
			body = r.Body
		}
	} else {
		// Not chunked or applying chunked
		if r.Compressed && len(r.RawBody) > 0 {
			body = r.RawBody // Use compressed body
		} else {
			body = r.Body
		}
	}

	// Step 2: Handle decompression if body was compressed
	if r.Compressed && opts.Compression == CompressionNone {
		// Need to decompress
		// If we started with RawBody (compressed), decompress it
		if len(r.Body) > 0 && !r.IsBodyChunked {
			body = r.Body // Use already decompressed body
		} else if len(body) > 0 {
			contentEncoding := r.GetContentEncoding()
			compType := compression.DetectCompression(contentEncoding)
			if compType != compression.CompressionNone {
				decompressed, err := compression.Decompress(body, compType)
				if err != nil {
					return nil, fmt.Errorf("decompression failed: %w", err)
				}
				body = decompressed
			}
		}
	}

	// Step 3: Handle recompression if requested
	if opts.Compression != CompressionKeep && opts.Compression != CompressionNone {
		// First ensure body is decompressed
		if r.Compressed && len(r.Body) > 0 {
			body = r.Body // Use decompressed body as source
		}

		// Apply new compression
		var compType compression.CompressionType
		switch opts.Compression {
		case CompressionGzip:
			compType = compression.CompressionGzip
		case CompressionDeflate:
			compType = compression.CompressionDeflate
		case CompressionBrotli:
			compType = compression.CompressionBrotli
		}

		compressed, err := compression.Compress(body, compType)
		if err != nil {
			return nil, fmt.Errorf("compression failed: %w", err)
		}
		body = compressed
	}

	// Step 4: Apply chunked encoding if requested
	if opts.Chunked == ChunkedApply && !r.IsBodyChunked {
		chunkSize := opts.ChunkSize
		if chunkSize <= 0 {
			chunkSize = 8192
		}
		body = chunked.Encode(body, chunkSize)
	}

	return body, nil
}

// prepareHeaders creates headers based on options
func (r *Response) prepareHeaders(opts BuildOptions, body []byte) []headerForBuild {
	var headers []headerForBuild

	// Determine what compression/chunked state the final body has
	finalCompression := r.determineCompression(opts)
	finalChunked := r.determineChunked(opts)

	for _, h := range r.Headers.All() {
		nameLower := strings.ToLower(h.Name)

		// Handle Content-Encoding
		if nameLower == "content-encoding" {
			if opts.UpdateContentEncoding {
				if finalCompression == CompressionNone {
					// Skip header (remove it)
					continue
				} else if finalCompression != CompressionKeep {
					// Update with new compression type
					headers = append(headers, headerForBuild{
						Name:         h.Name,
						Value:        compressionToString(finalCompression),
						OriginalLine: "",
						LineEnding:   h.LineEnding,
					})
					continue
				}
			}
		}

		// Handle Transfer-Encoding
		if nameLower == "transfer-encoding" {
			if opts.UpdateTransferEncoding {
				if finalChunked == ChunkedRemove {
					// Remove chunked from transfer-encoding
					newValue := removeChunkedFromTE(h.Value)
					if newValue == "" {
						continue // Skip header entirely
					}
					headers = append(headers, headerForBuild{
						Name:         h.Name,
						Value:        newValue,
						OriginalLine: "",
						LineEnding:   h.LineEnding,
					})
					continue
				} else if finalChunked == ChunkedApply && !r.IsBodyChunked {
					// Add chunked to transfer-encoding
					headers = append(headers, headerForBuild{
						Name:         h.Name,
						Value:        addChunkedToTE(h.Value),
						OriginalLine: "",
						LineEnding:   h.LineEnding,
					})
					continue
				}
			}
		}

		// Handle Content-Length
		if nameLower == "content-length" {
			if opts.UpdateContentLength {
				if finalChunked == ChunkedApply || (r.IsBodyChunked && finalChunked == ChunkedKeep) {
					// Chunked encoding doesn't use Content-Length
					continue
				}
				headers = append(headers, headerForBuild{
					Name:         h.Name,
					Value:        fmt.Sprintf("%d", len(body)),
					OriginalLine: "",
					LineEnding:   h.LineEnding,
				})
				continue
			}
		}

		// Keep header as-is
		headers = append(headers, headerForBuild{
			Name:         h.Name,
			Value:        h.Value,
			OriginalLine: h.OriginalLine,
			LineEnding:   h.LineEnding,
		})
	}

	// Add Transfer-Encoding: chunked if needed and not present
	if opts.UpdateTransferEncoding && finalChunked == ChunkedApply && !r.IsBodyChunked {
		hasTE := false
		for _, h := range headers {
			if strings.ToLower(h.Name) == "transfer-encoding" {
				hasTE = true
				break
			}
		}
		if !hasTE {
			headers = append(headers, headerForBuild{
				Name:       "Transfer-Encoding",
				Value:      "chunked",
				LineEnding: "\r\n",
			})
		}
	}

	// Add Content-Encoding if applying compression and not present
	if opts.UpdateContentEncoding && finalCompression != CompressionNone && finalCompression != CompressionKeep {
		hasCE := false
		for _, h := range headers {
			if strings.ToLower(h.Name) == "content-encoding" {
				hasCE = true
				break
			}
		}
		if !hasCE {
			headers = append(headers, headerForBuild{
				Name:       "Content-Encoding",
				Value:      compressionToString(finalCompression),
				LineEnding: "\r\n",
			})
		}
	}

	// Add Content-Length if needed and not present
	if opts.UpdateContentLength && finalChunked != ChunkedApply {
		hasCL := false
		for _, h := range headers {
			if strings.ToLower(h.Name) == "content-length" {
				hasCL = true
				break
			}
		}
		if !hasCL && len(body) > 0 {
			headers = append(headers, headerForBuild{
				Name:       "Content-Length",
				Value:      fmt.Sprintf("%d", len(body)),
				LineEnding: "\r\n",
			})
		}
	}

	return headers
}

// headerForBuild is a temporary header structure for building
type headerForBuild struct {
	Name         string
	Value        string
	OriginalLine string
	LineEnding   string
}

// determineCompression returns the final compression state
func (r *Response) determineCompression(opts BuildOptions) CompressionMethod {
	if opts.Compression != CompressionKeep {
		return opts.Compression
	}
	if r.Compressed {
		// Detect original compression
		ce := r.GetContentEncoding()
		switch strings.ToLower(ce) {
		case "gzip":
			return CompressionGzip
		case "deflate":
			return CompressionDeflate
		case "br":
			return CompressionBrotli
		}
	}
	return CompressionNone
}

// determineChunked returns the final chunked state
func (r *Response) determineChunked(opts BuildOptions) ChunkedOption {
	if opts.Chunked != ChunkedKeep {
		return opts.Chunked
	}
	if r.IsBodyChunked {
		return ChunkedApply
	}
	return ChunkedRemove
}

// buildHTTP1Format builds HTTP/1.x format output
func (r *Response) buildHTTP1Format(headers []headerForBuild, body []byte, lineSep string, preserveFormat bool) []byte {
	var buf bytes.Buffer

	// Status line
	buf.WriteString(r.Version)
	buf.WriteString(" ")
	buf.WriteString(fmt.Sprintf("%d", r.StatusCode))
	buf.WriteString(" ")
	buf.WriteString(r.StatusText)
	buf.WriteString(lineSep)

	// Headers
	for _, h := range headers {
		if preserveFormat && h.OriginalLine != "" {
			buf.WriteString(h.OriginalLine)
		} else {
			buf.WriteString(h.Name)
			buf.WriteString(": ")
			buf.WriteString(h.Value)
		}
		if h.LineEnding != "" {
			buf.WriteString(h.LineEnding)
		} else {
			buf.WriteString(lineSep)
		}
	}

	// Empty line
	buf.WriteString(lineSep)

	// Body
	if len(body) > 0 {
		buf.Write(body)
	}

	return buf.Bytes()
}

// buildHTTP2Format builds HTTP/2 style format output
func (r *Response) buildHTTP2Format(headers []headerForBuild, body []byte, lineSep string) []byte {
	var buf bytes.Buffer

	// Pseudo-header :status
	buf.WriteString(":status: ")
	buf.WriteString(fmt.Sprintf("%d", r.StatusCode))
	buf.WriteString(lineSep)

	// Regular headers (skip connection-specific ones)
	for _, h := range headers {
		nameLower := strings.ToLower(h.Name)
		// Skip HTTP/1.x specific headers
		if nameLower == "connection" ||
			nameLower == "keep-alive" ||
			nameLower == "proxy-connection" ||
			nameLower == "transfer-encoding" ||
			nameLower == "upgrade" {
			continue
		}
		buf.WriteString(h.Name)
		buf.WriteString(": ")
		buf.WriteString(h.Value)
		buf.WriteString(lineSep)
	}

	// Empty line
	buf.WriteString(lineSep)

	// Body
	if len(body) > 0 {
		buf.Write(body)
	}

	return buf.Bytes()
}

// compressionToString converts CompressionMethod to Content-Encoding string
func compressionToString(cm CompressionMethod) string {
	switch cm {
	case CompressionGzip:
		return "gzip"
	case CompressionDeflate:
		return "deflate"
	case CompressionBrotli:
		return "br"
	default:
		return ""
	}
}

// removeChunkedFromTE removes "chunked" from Transfer-Encoding value
func removeChunkedFromTE(te string) string {
	parts := strings.Split(te, ",")
	var filtered []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.ToLower(p) != "chunked" {
			filtered = append(filtered, p)
		}
	}
	return strings.Join(filtered, ", ")
}

// addChunkedToTE adds "chunked" to Transfer-Encoding value
func addChunkedToTE(te string) string {
	te = strings.TrimSpace(te)
	if te == "" {
		return "chunked"
	}
	return te + ", chunked"
}

// ============================================================================
// Convenience Methods
// ============================================================================

// IsCompressed returns true if the response body is compressed
func (r *Response) IsCompressed() bool {
	return r.Compressed
}

// IsChunked returns true if the response body is chunked encoded
func (r *Response) IsChunked() bool {
	return r.IsBodyChunked
}

// GetCompressionType returns the compression type of the response
func (r *Response) GetCompressionType() compression.CompressionType {
	if !r.Compressed {
		return compression.CompressionNone
	}
	return compression.DetectCompression(r.GetContentEncoding())
}

// BuildNormalized builds a normalized HTTP/1.1 response
// - Decompressed body
// - Dechunked body
// - Standard CRLF line endings
// - Updated headers
func (r *Response) BuildNormalized() ([]byte, error) {
	return r.BuildWithOptions(NormalizedOptions())
}

// BuildAsHTTP2 builds the response in HTTP/2 format
func (r *Response) BuildAsHTTP2() ([]byte, error) {
	return r.BuildWithOptions(HTTP2Options())
}

// BuildWithCompression builds with specified compression
func (r *Response) BuildWithCompression(cm CompressionMethod) ([]byte, error) {
	opts := DefaultBuildOptions()
	opts.Compression = cm
	return r.BuildWithOptions(opts)
}

// BuildDechunked builds with chunked encoding removed
func (r *Response) BuildDechunked() ([]byte, error) {
	opts := DefaultBuildOptions()
	opts.Chunked = ChunkedRemove
	return r.BuildWithOptions(opts)
}
