package compression

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/errors"
	"github.com/andybalholm/brotli"
)

// CompressionType represents supported compression algorithms
type CompressionType int

const (
	CompressionNone CompressionType = iota
	CompressionGzip
	CompressionDeflate
	CompressionBrotli
)

// DetectCompression detects compression type from Content-Encoding header
func DetectCompression(contentEncoding string) CompressionType {
	encoding := strings.ToLower(strings.TrimSpace(contentEncoding))

	switch encoding {
	case "gzip":
		return CompressionGzip
	case "deflate":
		return CompressionDeflate
	case "br", "brotli":
		return CompressionBrotli
	default:
		return CompressionNone
	}
}

// CompressionTypeToString converts a CompressionType to its Content-Encoding string
func CompressionTypeToString(ct CompressionType) string {
	switch ct {
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

// Decompress decompresses data based on the compression type
func Decompress(data []byte, compressionType CompressionType) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	switch compressionType {
	case CompressionGzip:
		return decompressGzip(data)
	case CompressionDeflate:
		return decompressDeflate(data)
	case CompressionBrotli:
		return decompressBrotli(data)
	case CompressionNone:
		return data, nil
	default:
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"unsupported compression type", "decompress", data)
	}
}

// decompressGzip decompresses gzip data
func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to create gzip reader: "+err.Error(), "decompressGzip", data)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to decompress gzip data: "+err.Error(), "decompressGzip", data)
	}

	return decompressed, nil
}

// decompressDeflate decompresses deflate data
func decompressDeflate(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to decompress deflate data: "+err.Error(), "decompressDeflate", data)
	}

	return decompressed, nil
}

// decompressBrotli decompresses brotli data
func decompressBrotli(data []byte) ([]byte, error) {
	reader := brotli.NewReader(bytes.NewReader(data))

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to decompress brotli data: "+err.Error(), "decompressBrotli", data)
	}

	return decompressed, nil
}

// Compress compresses data using the specified algorithm
func Compress(data []byte, compressionType CompressionType) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	switch compressionType {
	case CompressionGzip:
		return compressGzip(data)
	case CompressionDeflate:
		return compressDeflate(data)
	case CompressionBrotli:
		return compressBrotli(data)
	case CompressionNone:
		return data, nil
	default:
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"unsupported compression type", "compress", data)
	}
}

// compressGzip compresses data using gzip
func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to write gzip data: "+err.Error(), "compressGzip", data)
	}

	if err := writer.Close(); err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to close gzip writer: "+err.Error(), "compressGzip", data)
	}

	return buf.Bytes(), nil
}

// compressDeflate compresses data using deflate
func compressDeflate(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to create deflate writer: "+err.Error(), "compressDeflate", data)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to write deflate data: "+err.Error(), "compressDeflate", data)
	}

	if err := writer.Close(); err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to close deflate writer: "+err.Error(), "compressDeflate", data)
	}

	return buf.Bytes(), nil
}

// compressBrotli compresses data using brotli
func compressBrotli(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := brotli.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to write brotli data: "+err.Error(), "compressBrotli", data)
	}

	if err := writer.Close(); err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to close brotli writer: "+err.Error(), "compressBrotli", data)
	}

	return buf.Bytes(), nil
}
