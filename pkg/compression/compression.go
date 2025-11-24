package compression

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"strings"

	"github.com/WhileEndless/go-httptools/pkg/errors"
	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// CompressionType represents supported compression algorithms
type CompressionType int

const (
	CompressionNone CompressionType = iota
	CompressionGzip
	CompressionDeflate
	CompressionBrotli
	CompressionZstd
)

// DetectCompression detects compression type from Content-Encoding header
// Supports: gzip, x-gzip, deflate, br, brotli, zstd, identity
func DetectCompression(contentEncoding string) CompressionType {
	encoding := strings.ToLower(strings.TrimSpace(contentEncoding))

	switch encoding {
	case "gzip", "x-gzip":
		return CompressionGzip
	case "deflate", "x-deflate":
		return CompressionDeflate
	case "br", "brotli":
		return CompressionBrotli
	case "zstd", "zstandard":
		return CompressionZstd
	case "identity", "":
		return CompressionNone
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
	case CompressionZstd:
		return "zstd"
	default:
		return ""
	}
}

// IsSupported checks if a Content-Encoding value is supported
func IsSupported(contentEncoding string) bool {
	encoding := strings.ToLower(strings.TrimSpace(contentEncoding))
	switch encoding {
	case "gzip", "x-gzip", "deflate", "x-deflate", "br", "brotli", "zstd", "zstandard", "identity", "":
		return true
	default:
		return false
	}
}

// GetSupportedEncodings returns a list of supported Content-Encoding values
func GetSupportedEncodings() []string {
	return []string{"gzip", "deflate", "br", "zstd", "identity"}
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
	case CompressionZstd:
		return decompressZstd(data)
	case CompressionNone:
		return data, nil
	default:
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"unsupported compression type", "decompress", data)
	}
}

// DecompressAuto automatically detects and decompresses data
// Tries to detect compression from magic bytes if Content-Encoding is unknown
func DecompressAuto(data []byte) ([]byte, CompressionType, error) {
	if len(data) == 0 {
		return data, CompressionNone, nil
	}

	// Try to detect by magic bytes
	ct := DetectByMagicBytes(data)
	if ct == CompressionNone {
		return data, CompressionNone, nil
	}

	decompressed, err := Decompress(data, ct)
	if err != nil {
		return nil, CompressionNone, err
	}
	return decompressed, ct, nil
}

// DetectByMagicBytes attempts to detect compression type from data magic bytes
func DetectByMagicBytes(data []byte) CompressionType {
	if len(data) < 2 {
		return CompressionNone
	}

	// Gzip: 1f 8b
	if data[0] == 0x1f && data[1] == 0x8b {
		return CompressionGzip
	}

	// Zstd: 28 b5 2f fd
	if len(data) >= 4 && data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd {
		return CompressionZstd
	}

	// Deflate: commonly starts with 78 (9c, da, 5e, 01)
	if data[0] == 0x78 && (data[1] == 0x9c || data[1] == 0xda || data[1] == 0x5e || data[1] == 0x01) {
		return CompressionDeflate
	}

	// Brotli doesn't have a fixed magic number, harder to detect
	// Check for common brotli stream header patterns
	// First nibble is window size (0-11), second nibble indicates type
	if len(data) >= 1 {
		firstByte := data[0]
		// Brotli streams often start with specific patterns
		// This is a heuristic, not guaranteed
		windowBits := firstByte & 0x0F
		if windowBits <= 0x0B {
			// Could be brotli, but we can't be sure without trying to decompress
			// Skip auto-detection for brotli
		}
	}

	return CompressionNone
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

// decompressZstd decompresses zstd data
func decompressZstd(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to create zstd reader: "+err.Error(), "decompressZstd", data)
	}
	defer decoder.Close()

	decompressed, err := io.ReadAll(decoder)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to decompress zstd data: "+err.Error(), "decompressZstd", data)
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
	case CompressionZstd:
		return compressZstd(data)
	case CompressionNone:
		return data, nil
	default:
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"unsupported compression type", "compress", data)
	}
}

// CompressWithLevel compresses data with specified compression level
// Level interpretation varies by algorithm:
// - gzip/deflate: 1-9 (1=fastest, 9=best)
// - brotli: 0-11 (0=fastest, 11=best)
// - zstd: 1-22 (1=fastest, 22=best), 0=default
func CompressWithLevel(data []byte, compressionType CompressionType, level int) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	switch compressionType {
	case CompressionGzip:
		return compressGzipLevel(data, level)
	case CompressionDeflate:
		return compressDeflateLevel(data, level)
	case CompressionBrotli:
		return compressBrotliLevel(data, level)
	case CompressionZstd:
		return compressZstdLevel(data, level)
	case CompressionNone:
		return data, nil
	default:
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"unsupported compression type", "compressWithLevel", data)
	}
}

// compressGzip compresses data using gzip (default level)
func compressGzip(data []byte) ([]byte, error) {
	return compressGzipLevel(data, gzip.DefaultCompression)
}

// compressGzipLevel compresses data using gzip with specified level
func compressGzipLevel(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to create gzip writer: "+err.Error(), "compressGzip", data)
	}

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

// compressDeflate compresses data using deflate (default level)
func compressDeflate(data []byte) ([]byte, error) {
	return compressDeflateLevel(data, flate.DefaultCompression)
}

// compressDeflateLevel compresses data using deflate with specified level
func compressDeflateLevel(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, level)
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

// compressBrotli compresses data using brotli (default level)
func compressBrotli(data []byte) ([]byte, error) {
	return compressBrotliLevel(data, brotli.DefaultCompression)
}

// compressBrotliLevel compresses data using brotli with specified level
func compressBrotliLevel(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer := brotli.NewWriterLevel(&buf, level)

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

// compressZstd compresses data using zstd (default level)
func compressZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to create zstd writer: "+err.Error(), "compressZstd", data)
	}
	defer encoder.Close()

	return encoder.EncodeAll(data, nil), nil
}

// compressZstdLevel compresses data using zstd with specified level
func compressZstdLevel(data []byte, level int) ([]byte, error) {
	// Map level to zstd.EncoderLevel
	var encLevel zstd.EncoderLevel
	switch {
	case level <= 0:
		encLevel = zstd.SpeedDefault
	case level <= 3:
		encLevel = zstd.SpeedFastest
	case level <= 6:
		encLevel = zstd.SpeedDefault
	case level <= 12:
		encLevel = zstd.SpeedBetterCompression
	default:
		encLevel = zstd.SpeedBestCompression
	}

	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(encLevel))
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeCompressionError,
			"failed to create zstd writer: "+err.Error(), "compressZstd", data)
	}
	defer encoder.Close()

	return encoder.EncodeAll(data, nil), nil
}
