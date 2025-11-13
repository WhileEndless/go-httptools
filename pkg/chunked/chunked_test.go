package chunked

import (
	"bytes"
	"testing"
)

func TestDecode_Simple(t *testing.T) {
	input := []byte("3\r\nfoo\r\n3\r\nbar\r\n0\r\n\r\n")
	body, trailers := Decode(input)

	expected := "foobar"
	if string(body) != expected {
		t.Errorf("Expected body %q, got %q", expected, string(body))
	}

	if len(trailers) != 0 {
		t.Errorf("Expected no trailers, got %d", len(trailers))
	}
}

func TestDecode_WithTrailers(t *testing.T) {
	input := []byte("3\r\nfoo\r\n0\r\nX-Checksum: abc123\r\nX-Custom: value\r\n\r\n")
	body, trailers := Decode(input)

	if string(body) != "foo" {
		t.Errorf("Expected body %q, got %q", "foo", string(body))
	}

	if len(trailers) != 2 {
		t.Errorf("Expected 2 trailers, got %d", len(trailers))
	}

	if trailers["X-Checksum"] != "abc123" {
		t.Errorf("Expected trailer X-Checksum=abc123, got %q", trailers["X-Checksum"])
	}

	if trailers["X-Custom"] != "value" {
		t.Errorf("Expected trailer X-Custom=value, got %q", trailers["X-Custom"])
	}
}

func TestDecode_UnixLineEndings(t *testing.T) {
	input := []byte("3\nfoo\n3\nbar\n0\n\n")
	body, _ := Decode(input)

	expected := "foobar"
	if string(body) != expected {
		t.Errorf("Expected body %q, got %q", expected, string(body))
	}
}

func TestDecode_ChunkExtensions(t *testing.T) {
	// Chunk extensions (e.g., "5;name=value") should be ignored
	input := []byte("3;ext=val\r\nfoo\r\n3;another\r\nbar\r\n0\r\n\r\n")
	body, _ := Decode(input)

	expected := "foobar"
	if string(body) != expected {
		t.Errorf("Expected body %q, got %q", expected, string(body))
	}
}

func TestDecode_Empty(t *testing.T) {
	input := []byte("0\r\n\r\n")
	body, trailers := Decode(input)

	if len(body) != 0 {
		t.Errorf("Expected empty body, got %q", string(body))
	}

	if len(trailers) != 0 {
		t.Errorf("Expected no trailers, got %d", len(trailers))
	}
}

func TestDecode_EmptyInput(t *testing.T) {
	input := []byte("")
	body, trailers := Decode(input)

	if len(body) != 0 {
		t.Errorf("Expected empty body, got %q", string(body))
	}

	if len(trailers) != 0 {
		t.Errorf("Expected no trailers, got %d", len(trailers))
	}
}

// Fault tolerance tests - malformed input should not panic

func TestDecode_Malformed_NoLineEnding(t *testing.T) {
	input := []byte("3foobar")
	body, _ := Decode(input) // Should not panic
	// Best effort: may return empty or partial data
	_ = body
}

func TestDecode_Malformed_InvalidHex(t *testing.T) {
	input := []byte("ZZZ\r\ndata\r\n0\r\n\r\n")
	body, _ := Decode(input) // Should not panic
	// Best effort: stops at invalid chunk
	_ = body
}

func TestDecode_Malformed_NegativeSize(t *testing.T) {
	input := []byte("-5\r\ndata\r\n0\r\n\r\n")
	body, _ := Decode(input) // Should not panic
	_ = body
}

func TestDecode_Malformed_InsufficientData(t *testing.T) {
	input := []byte("a\r\nfoo\r\n") // Claims 10 bytes but only has 3
	body, _ := Decode(input)       // Should not panic
	// Best effort: takes what's available (may include trailing CRLF)
	if len(body) == 0 {
		t.Error("Expected best-effort parse to return some data")
	}
}

func TestDecode_Malformed_MissingTrailingCRLF(t *testing.T) {
	input := []byte("3\r\nfoo")
	body, _ := Decode(input) // Should not panic
	// Best effort
	_ = body
}

func TestEncode_Simple(t *testing.T) {
	input := []byte("foobar")
	encoded := Encode(input, 3)

	expected := []byte("3\r\nfoo\r\n3\r\nbar\r\n0\r\n\r\n")
	if !bytes.Equal(encoded, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(encoded))
	}
}

func TestEncode_SingleChunk(t *testing.T) {
	input := []byte("hello")
	encoded := Encode(input, 100) // Chunk size larger than data

	expected := []byte("5\r\nhello\r\n0\r\n\r\n")
	if !bytes.Equal(encoded, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(encoded))
	}
}

func TestEncode_Empty(t *testing.T) {
	input := []byte("")
	encoded := Encode(input, 10)

	expected := []byte("0\r\n\r\n")
	if !bytes.Equal(encoded, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(encoded))
	}
}

func TestEncode_DefaultChunkSize(t *testing.T) {
	input := []byte("test")
	encoded := Encode(input, 0) // Should use default chunk size

	// Should still encode correctly
	decoded, _ := Decode(encoded)
	if !bytes.Equal(decoded, input) {
		t.Errorf("Round-trip failed: expected %q, got %q", string(input), string(decoded))
	}
}

func TestEncodeWithTrailers(t *testing.T) {
	input := []byte("foo")
	trailers := map[string]string{
		"X-Checksum": "abc123",
		"X-Custom":   "value",
	}

	encoded := EncodeWithTrailers(input, 3, trailers)
	decoded, decodedTrailers := Decode(encoded)

	if string(decoded) != "foo" {
		t.Errorf("Expected body %q, got %q", "foo", string(decoded))
	}

	if len(decodedTrailers) != 2 {
		t.Errorf("Expected 2 trailers, got %d", len(decodedTrailers))
	}
}

func TestRoundTrip(t *testing.T) {
	testCases := [][]byte{
		[]byte("Hello, World!"),
		[]byte(""),
		[]byte("a"),
		[]byte("This is a longer message that spans multiple chunks"),
		[]byte("Special chars: \r\n\t\x00\xFF"),
	}

	for _, original := range testCases {
		encoded := Encode(original, 5)
		decoded, _ := Decode(encoded)

		if !bytes.Equal(original, decoded) {
			t.Errorf("Round-trip failed for %q: got %q", string(original), string(decoded))
		}
	}
}

func TestRoundTrip_VariousChunkSizes(t *testing.T) {
	data := []byte("The quick brown fox jumps over the lazy dog")
	chunkSizes := []int{1, 3, 5, 10, 100, 1000}

	for _, chunkSize := range chunkSizes {
		encoded := Encode(data, chunkSize)
		decoded, _ := Decode(encoded)

		if !bytes.Equal(data, decoded) {
			t.Errorf("Round-trip failed with chunk size %d", chunkSize)
		}
	}
}

func TestIsChunked_Valid(t *testing.T) {
	validCases := [][]byte{
		[]byte("3\r\nfoo\r\n0\r\n\r\n"),
		[]byte("a\r\n1234567890\r\n0\r\n\r\n"),
		[]byte("10\r\nabcdefghijklmnop\r\n0\r\n\r\n"),
		[]byte("5\nhello\n0\n\n"), // Unix line endings
	}

	for i, input := range validCases {
		if !IsChunked(input) {
			t.Errorf("Case %d: Expected IsChunked=true for %q", i, string(input[:min(20, len(input))]))
		}
	}
}

func TestIsChunked_Invalid(t *testing.T) {
	invalidCases := [][]byte{
		[]byte(""),
		[]byte("ab"),
		[]byte("not chunked data"),
		[]byte("HTTP/1.1 200 OK\r\n"),
		[]byte("GET / HTTP/1.1\r\n"),
	}

	for i, input := range invalidCases {
		if IsChunked(input) {
			t.Errorf("Case %d: Expected IsChunked=false for %q", i, string(input))
		}
	}
}

func TestIsChunked_EdgeCases(t *testing.T) {
	// Very short input
	if IsChunked([]byte("a")) {
		t.Error("Expected false for very short input")
	}

	// No line ending
	if IsChunked([]byte("123456789012345678901234567890")) {
		t.Error("Expected false for input without line ending")
	}

	// Line too long
	if IsChunked([]byte("12345678901234567890\r\n")) {
		t.Error("Expected false for very long first line")
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Benchmark tests
func BenchmarkDecode(b *testing.B) {
	input := []byte("3\r\nfoo\r\n3\r\nbar\r\n3\r\nbaz\r\n0\r\n\r\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(input)
	}
}

func BenchmarkEncode(b *testing.B) {
	input := []byte("foobar")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Encode(input, 3)
	}
}

func BenchmarkIsChunked(b *testing.B) {
	input := []byte("5\r\nhello\r\n0\r\n\r\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsChunked(input)
	}
}
