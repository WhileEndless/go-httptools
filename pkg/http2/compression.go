package http2

import (
	"strconv"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/compression"
)

// ============================================================================
// HTTP/2 Response Compression Support
// ============================================================================

// CompressBody compresses the body using the specified compression type
// Updates Content-Encoding header and sets RawBody with compressed data
func (r *Response) CompressBody(compressionType compression.CompressionType) error {
	if len(r.Body) == 0 {
		return nil
	}

	compressed, err := compression.Compress(r.Body, compressionType)
	if err != nil {
		return err
	}

	r.RawBody = compressed
	r.Compressed = true

	// Update headers
	r.Headers.Set("content-encoding", compression.CompressionTypeToString(compressionType))
	r.Headers.Set("content-length", strconv.Itoa(len(compressed)))

	return nil
}

// DecompressBody decompresses the body if it's compressed
// Updates Body with decompressed data
func (r *Response) DecompressBody() error {
	if !r.Compressed || len(r.RawBody) == 0 {
		return nil
	}

	contentEncoding := strings.TrimSpace(r.Headers.Get("content-encoding"))
	if contentEncoding == "" {
		return nil
	}

	compressionType := compression.DetectCompression(contentEncoding)
	if compressionType == compression.CompressionNone {
		return nil
	}

	decompressed, err := compression.Decompress(r.RawBody, compressionType)
	if err != nil {
		return err
	}

	r.Body = decompressed
	r.Compressed = false

	return nil
}

// GetDecompressedBody returns decompressed body without modifying the response
func (r *Response) GetDecompressedBody() ([]byte, error) {
	if !r.Compressed || len(r.RawBody) == 0 {
		return r.Body, nil
	}

	contentEncoding := strings.TrimSpace(r.Headers.Get("content-encoding"))
	if contentEncoding == "" {
		return r.Body, nil
	}

	compressionType := compression.DetectCompression(contentEncoding)
	if compressionType == compression.CompressionNone {
		return r.Body, nil
	}

	return compression.Decompress(r.RawBody, compressionType)
}

// BuildDecompressed builds response as HTTP/2 format with decompressed body
func (r *Response) BuildDecompressed() ([]byte, error) {
	// Get decompressed body
	body, err := r.GetDecompressedBody()
	if err != nil {
		return nil, err
	}

	// Create a temporary copy without compression headers
	clone := r.Clone()
	clone.Body = body
	clone.RawBody = nil
	clone.Compressed = false
	clone.Headers.Del("content-encoding")
	clone.Headers.Set("content-length", strconv.Itoa(len(body)))

	return clone.Build(), nil
}

// BuildAsHTTP1Decompressed builds as HTTP/1.1 with decompressed body
func (r *Response) BuildAsHTTP1Decompressed() ([]byte, error) {
	// Get decompressed body
	body, err := r.GetDecompressedBody()
	if err != nil {
		return nil, err
	}

	// Create a temporary copy without compression headers
	clone := r.Clone()
	clone.Body = body
	clone.RawBody = nil
	clone.Compressed = false
	clone.Headers.Del("content-encoding")
	clone.Headers.Set("content-length", strconv.Itoa(len(body)))

	return clone.BuildAsHTTP1(), nil
}

// ============================================================================
// HTTP/2 Request Compression Support
// ============================================================================

// CompressBody compresses the request body
func (r *Request) CompressBody(compressionType compression.CompressionType) error {
	if len(r.Body) == 0 {
		return nil
	}

	compressed, err := compression.Compress(r.Body, compressionType)
	if err != nil {
		return err
	}

	r.RawBody = r.Body
	r.Body = compressed

	// Update headers
	r.Headers.Set("content-encoding", compression.CompressionTypeToString(compressionType))
	r.Headers.Set("content-length", strconv.Itoa(len(compressed)))

	return nil
}

// DecompressBody decompresses the request body if compressed
func (r *Request) DecompressBody() error {
	if len(r.Body) == 0 {
		return nil
	}

	contentEncoding := strings.TrimSpace(r.Headers.Get("content-encoding"))
	if contentEncoding == "" {
		return nil
	}

	compressionType := compression.DetectCompression(contentEncoding)
	if compressionType == compression.CompressionNone {
		return nil
	}

	decompressed, err := compression.Decompress(r.Body, compressionType)
	if err != nil {
		return err
	}

	r.RawBody = r.Body
	r.Body = decompressed

	return nil
}

// GetDecompressedBody returns decompressed body without modifying the request
func (r *Request) GetDecompressedBody() ([]byte, error) {
	if len(r.Body) == 0 {
		return nil, nil
	}

	contentEncoding := strings.TrimSpace(r.Headers.Get("content-encoding"))
	if contentEncoding == "" {
		return r.Body, nil
	}

	compressionType := compression.DetectCompression(contentEncoding)
	if compressionType == compression.CompressionNone {
		return r.Body, nil
	}

	return compression.Decompress(r.Body, compressionType)
}

// BuildDecompressed builds request as HTTP/2 format with decompressed body
func (r *Request) BuildDecompressed() ([]byte, error) {
	body, err := r.GetDecompressedBody()
	if err != nil {
		return nil, err
	}

	clone := r.Clone()
	clone.Body = body
	clone.RawBody = nil
	clone.Headers.Del("content-encoding")
	clone.Headers.Set("content-length", strconv.Itoa(len(body)))

	return clone.Build(), nil
}

// BuildAsHTTP1Decompressed builds as HTTP/1.1 with decompressed body
func (r *Request) BuildAsHTTP1Decompressed() ([]byte, error) {
	body, err := r.GetDecompressedBody()
	if err != nil {
		return nil, err
	}

	clone := r.Clone()
	clone.Body = body
	clone.RawBody = nil
	clone.Headers.Del("content-encoding")
	clone.Headers.Set("content-length", strconv.Itoa(len(body)))

	return clone.BuildAsHTTP1(), nil
}
