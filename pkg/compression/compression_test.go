package compression

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"testing"

	"github.com/andybalholm/brotli"
)

// TestDetectCompression verifies compression type detection
func TestDetectCompression(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		expected CompressionType
	}{
		{"gzip lowercase", "gzip", CompressionGzip},
		{"gzip uppercase", "GZIP", CompressionGzip},
		{"gzip with spaces", "  gzip  ", CompressionGzip},
		{"deflate lowercase", "deflate", CompressionDeflate},
		{"deflate uppercase", "DEFLATE", CompressionDeflate},
		{"br lowercase", "br", CompressionBrotli},
		{"br uppercase", "BR", CompressionBrotli},
		{"brotli full name", "brotli", CompressionBrotli},
		{"brotli uppercase", "BROTLI", CompressionBrotli},
		{"unknown encoding", "zstd", CompressionNone},
		{"empty string", "", CompressionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectCompression(tt.encoding)
			if result != tt.expected {
				t.Errorf("DetectCompression(%q) = %v, expected %v",
					tt.encoding, result, tt.expected)
			}
		})
	}
}

// TestDecompressGzip verifies gzip decompression
func TestDecompressGzip(t *testing.T) {
	original := []byte("Hello, this is a test message for gzip compression!")

	// Compress with gzip
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(original); err != nil {
		t.Fatalf("Failed to write gzip data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}
	compressed := buf.Bytes()

	// Decompress using our function
	decompressed, err := decompressGzip(compressed)
	if err != nil {
		t.Fatalf("decompressGzip failed: %v", err)
	}

	// Verify
	if !bytes.Equal(decompressed, original) {
		t.Errorf("Decompressed data doesn't match original.\nExpected: %s\nGot: %s",
			string(original), string(decompressed))
	}
}

// TestDecompressDeflate verifies deflate decompression
func TestDecompressDeflate(t *testing.T) {
	original := []byte("Hello, this is a test message for deflate compression!")

	// Compress with deflate
	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create deflate writer: %v", err)
	}
	if _, err := writer.Write(original); err != nil {
		t.Fatalf("Failed to write deflate data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close deflate writer: %v", err)
	}
	compressed := buf.Bytes()

	// Decompress using our function
	decompressed, err := decompressDeflate(compressed)
	if err != nil {
		t.Fatalf("decompressDeflate failed: %v", err)
	}

	// Verify
	if !bytes.Equal(decompressed, original) {
		t.Errorf("Decompressed data doesn't match original.\nExpected: %s\nGot: %s",
			string(original), string(decompressed))
	}
}

// TestDecompressBrotli verifies brotli decompression
func TestDecompressBrotli(t *testing.T) {
	original := []byte("Hello, this is a test message for brotli compression!")

	// Compress with brotli
	var buf bytes.Buffer
	writer := brotli.NewWriter(&buf)
	if _, err := writer.Write(original); err != nil {
		t.Fatalf("Failed to write brotli data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close brotli writer: %v", err)
	}
	compressed := buf.Bytes()

	// Decompress using our function
	decompressed, err := decompressBrotli(compressed)
	if err != nil {
		t.Fatalf("decompressBrotli failed: %v", err)
	}

	// Verify
	if !bytes.Equal(decompressed, original) {
		t.Errorf("Decompressed data doesn't match original.\nExpected: %s\nGot: %s",
			string(original), string(decompressed))
	}
}

// TestDecompress verifies the main Decompress function
func TestDecompress(t *testing.T) {
	original := []byte("Test message for compression algorithms!")

	tests := []struct {
		name            string
		compressionType CompressionType
	}{
		{"gzip", CompressionGzip},
		{"deflate", CompressionDeflate},
		{"brotli", CompressionBrotli},
		{"none", CompressionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			compressed, err := Compress(original, tt.compressionType)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			// Decompress
			decompressed, err := Decompress(compressed, tt.compressionType)
			if err != nil {
				t.Fatalf("Decompress failed: %v", err)
			}

			// Verify
			if !bytes.Equal(decompressed, original) {
				t.Errorf("Decompressed data doesn't match original.\nExpected: %s\nGot: %s",
					string(original), string(decompressed))
			}
		})
	}
}

// TestCompressGzip verifies gzip compression
func TestCompressGzip(t *testing.T) {
	original := []byte("Hello, this is a test message for gzip compression!")

	// Compress using our function
	compressed, err := compressGzip(original)
	if err != nil {
		t.Fatalf("compressGzip failed: %v", err)
	}

	// Verify it can be decompressed with standard library
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("Failed to read gzip data: %v", err)
	}

	// Verify
	if !bytes.Equal(buf.Bytes(), original) {
		t.Errorf("Decompressed data doesn't match original.\nExpected: %s\nGot: %s",
			string(original), string(buf.Bytes()))
	}
}

// TestCompressDeflate verifies deflate compression
func TestCompressDeflate(t *testing.T) {
	original := []byte("Hello, this is a test message for deflate compression!")

	// Compress using our function
	compressed, err := compressDeflate(original)
	if err != nil {
		t.Fatalf("compressDeflate failed: %v", err)
	}

	// Verify it can be decompressed with standard library
	reader := flate.NewReader(bytes.NewReader(compressed))
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("Failed to read deflate data: %v", err)
	}

	// Verify
	if !bytes.Equal(buf.Bytes(), original) {
		t.Errorf("Decompressed data doesn't match original.\nExpected: %s\nGot: %s",
			string(original), string(buf.Bytes()))
	}
}

// TestCompressBrotli verifies brotli compression
func TestCompressBrotli(t *testing.T) {
	original := []byte("Hello, this is a test message for brotli compression!")

	// Compress using our function
	compressed, err := compressBrotli(original)
	if err != nil {
		t.Fatalf("compressBrotli failed: %v", err)
	}

	// Verify it can be decompressed with brotli library
	reader := brotli.NewReader(bytes.NewReader(compressed))

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("Failed to read brotli data: %v", err)
	}

	// Verify
	if !bytes.Equal(buf.Bytes(), original) {
		t.Errorf("Decompressed data doesn't match original.\nExpected: %s\nGot: %s",
			string(original), string(buf.Bytes()))
	}
}

// TestCompress verifies the main Compress function
func TestCompress(t *testing.T) {
	original := []byte("Test message for compression algorithms!")

	tests := []struct {
		name            string
		compressionType CompressionType
		shouldCompress  bool
	}{
		{"gzip", CompressionGzip, true},
		{"deflate", CompressionDeflate, true},
		{"brotli", CompressionBrotli, true},
		{"none", CompressionNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := Compress(original, tt.compressionType)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			if tt.shouldCompress {
				// Compressed data should be different (and usually smaller for this simple text)
				if bytes.Equal(compressed, original) {
					t.Error("Expected compressed data to be different from original")
				}
			} else {
				// CompressionNone should return original data unchanged
				if !bytes.Equal(compressed, original) {
					t.Error("Expected CompressionNone to return original data")
				}
			}
		})
	}
}

// TestRoundTrip verifies compress/decompress round trip for all algorithms
func TestRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"short text", []byte("Hello World!")},
		{"medium text", []byte("This is a longer test message that contains more text to compress. It has multiple sentences and should compress well with all algorithms.")},
		{"empty", []byte("")},
		{"json", []byte(`{"name":"test","values":[1,2,3,4,5],"nested":{"key":"value"}}`)},
		{"html", []byte(`<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hello</h1></body></html>`)},
	}

	compressionTypes := []struct {
		name string
		typ  CompressionType
	}{
		{"gzip", CompressionGzip},
		{"deflate", CompressionDeflate},
		{"brotli", CompressionBrotli},
	}

	for _, tc := range testCases {
		for _, ct := range compressionTypes {
			testName := tc.name + "_" + ct.name
			t.Run(testName, func(t *testing.T) {
				// Compress
				compressed, err := Compress(tc.data, ct.typ)
				if err != nil {
					t.Fatalf("Compress failed: %v", err)
				}

				// Decompress
				decompressed, err := Decompress(compressed, ct.typ)
				if err != nil {
					t.Fatalf("Decompress failed: %v", err)
				}

				// Verify
				if !bytes.Equal(decompressed, tc.data) {
					t.Errorf("Round trip failed.\nOriginal: %s\nDecompressed: %s",
						string(tc.data), string(decompressed))
				}
			})
		}
	}
}

// TestDecompressEmptyData verifies handling of empty data
func TestDecompressEmptyData(t *testing.T) {
	compressionTypes := []CompressionType{
		CompressionGzip,
		CompressionDeflate,
		CompressionBrotli,
		CompressionNone,
	}

	for _, ct := range compressionTypes {
		t.Run(ct.String(), func(t *testing.T) {
			decompressed, err := Decompress([]byte{}, ct)
			if err != nil {
				t.Fatalf("Decompress of empty data failed: %v", err)
			}
			if len(decompressed) != 0 {
				t.Errorf("Expected empty result, got %d bytes", len(decompressed))
			}
		})
	}
}

// TestCompressEmptyData verifies handling of empty data
func TestCompressEmptyData(t *testing.T) {
	compressionTypes := []CompressionType{
		CompressionGzip,
		CompressionDeflate,
		CompressionBrotli,
		CompressionNone,
	}

	for _, ct := range compressionTypes {
		t.Run(ct.String(), func(t *testing.T) {
			compressed, err := Compress([]byte{}, ct)
			if err != nil {
				t.Fatalf("Compress of empty data failed: %v", err)
			}
			if len(compressed) != 0 {
				t.Errorf("Expected empty result, got %d bytes", len(compressed))
			}
		})
	}
}

// Helper function to convert CompressionType to string for testing
func (ct CompressionType) String() string {
	switch ct {
	case CompressionGzip:
		return "gzip"
	case CompressionDeflate:
		return "deflate"
	case CompressionBrotli:
		return "brotli"
	case CompressionNone:
		return "none"
	default:
		return "unknown"
	}
}

// TestDecompressInvalidGzip verifies error handling for invalid gzip data
func TestDecompressInvalidGzip(t *testing.T) {
	invalidData := []byte("this is not valid gzip data")
	_, err := decompressGzip(invalidData)
	if err == nil {
		t.Error("Expected error for invalid gzip data")
	}
}

// TestDecompressInvalidDeflate verifies error handling for invalid deflate data
func TestDecompressInvalidDeflate(t *testing.T) {
	invalidData := []byte("this is not valid deflate data")
	_, err := decompressDeflate(invalidData)
	if err == nil {
		t.Error("Expected error for invalid deflate data")
	}
}

// TestDecompressInvalidBrotli verifies error handling for invalid brotli data
func TestDecompressInvalidBrotli(t *testing.T) {
	invalidData := []byte("this is not valid brotli data")
	_, err := decompressBrotli(invalidData)
	if err == nil {
		t.Error("Expected error for invalid brotli data")
	}
}
