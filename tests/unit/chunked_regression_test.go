package unit

import (
	"strings"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/response"
)

// Regression tests for chunked transfer encoding parsing
// These tests ensure that the parser correctly handles edge cases
// that could lead to data loss or corruption, particularly:
// - Newline characters within chunk data
// - Multiple chunks with embedded newlines
// - Binary data containing 0x0A (newline) bytes

// TestChunkedParsingWithNewlinesInData tests the specific bug scenario
// where chunk data contains newline characters that should NOT be treated
// as line breaks by the parser
func TestChunkedParsingWithNewlinesInData(t *testing.T) {
	// This simulates a chunked response with 2 chunks where each chunk's
	// data contains a newline character (\n) at the end (like JSON streaming APIs)
	chunk1 := `{"id": 0, "data": "first chunk"}` + "\n"  // 35 bytes
	chunk2 := `{"id": 1, "data": "second chunk"}` + "\n" // 36 bytes

	rawResponse := "HTTP/1.1 200 OK\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		"23\r\n" + chunk1 + "\r\n" + // 0x23 = 35 bytes
		"24\r\n" + chunk2 + "\r\n" + // 0x24 = 36 bytes
		"0\r\n" +
		"\r\n"

	// First, test with default parsing (no auto-decode)
	htResp, err := response.Parse([]byte(rawResponse))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should detect chunked encoding
	if !htResp.IsBodyChunked {
		t.Error("Expected IsBodyChunked=true")
	}

	// The body should contain the FULL chunked data, not just the first chunk
	bodyStr := string(htResp.Body)

	// Both chunk size declarations should be present
	if !strings.Contains(bodyStr, "23") {
		t.Error("Expected first chunk size marker '23' in raw chunked body")
	}
	if !strings.Contains(bodyStr, "24") {
		t.Error("Expected second chunk size marker '24' in raw chunked body - THIS INDICATES THE BUG!")
	}

	t.Logf("Raw chunked body length: %d bytes", len(htResp.Body))
	t.Logf("Raw chunked body:\n%s", bodyStr)

	// Now test with auto-decode
	opts := response.ParseOptions{
		AutoDecodeChunked: true,
	}

	htResp2, err := response.ParseWithOptions([]byte(rawResponse), opts)
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	// After decoding, both JSON objects should be present
	decodedBody := string(htResp2.Body)

	if !strings.Contains(decodedBody, `"id": 0`) {
		t.Error("Expected first JSON object with id=0 in decoded body")
	}
	if !strings.Contains(decodedBody, `"id": 1`) {
		t.Error("Expected second JSON object with id=1 in decoded body - THIS INDICATES THE BUG!")
	}

	// Expected decoded body should be both chunks concatenated
	// Chunk 1: {"id": 0, "data": "first chunk"}\n (35 bytes)
	// Chunk 2: {"id": 1, "data": "second chunk"}\n (36 bytes)
	// Total: 71 bytes
	expectedDecodedLength := 35 + 36 // 71 bytes
	actualDecodedLength := len(htResp2.Body)

	if actualDecodedLength != expectedDecodedLength {
		t.Errorf("Expected decoded body length %d bytes, got %d bytes - DATA LOSS DETECTED!",
			expectedDecodedLength, actualDecodedLength)
	}

	t.Logf("Decoded body length: %d bytes", actualDecodedLength)
	t.Logf("Decoded body:\n%s", decodedBody)
}

// TestChunkedParsingMultipleChunksWithEmbeddedNewlines tests multiple chunks
// each containing newline characters in their data (simulating streaming JSON)
func TestChunkedParsingMultipleChunksWithEmbeddedNewlines(t *testing.T) {
	// Simulates https://nghttp2.org/httpbin/stream/5 which sends 5 JSON objects
	// Each JSON object ends with \n
	chunk1 := `{"id": 0, "url": "https://example.com/0"}` + "\n" // 43 bytes
	chunk2 := `{"id": 1, "url": "https://example.com/1"}` + "\n" // 43 bytes
	chunk3 := `{"id": 2, "url": "https://example.com/2"}` + "\n" // 43 bytes
	chunk4 := `{"id": 3, "url": "https://example.com/3"}` + "\n" // 43 bytes
	chunk5 := `{"id": 4, "url": "https://example.com/4"}` + "\n" // 43 bytes

	// Build chunked response
	rawResponse := "HTTP/1.1 200 OK\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		"2b\r\n" + chunk1 + "\r\n" + // 0x2b = 43
		"2b\r\n" + chunk2 + "\r\n" +
		"2b\r\n" + chunk3 + "\r\n" +
		"2b\r\n" + chunk4 + "\r\n" +
		"2b\r\n" + chunk5 + "\r\n" +
		"0\r\n" +
		"\r\n"

	opts := response.ParseOptions{
		AutoDecodeChunked: true,
	}

	htResp, err := response.ParseWithOptions([]byte(rawResponse), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// All 5 JSON objects should be present
	decodedBody := string(htResp.Body)

	for i := 0; i < 5; i++ {
		expectedStr := `"id": ` + string(rune('0'+i))
		if !strings.Contains(decodedBody, expectedStr) {
			t.Errorf("Expected JSON object with id=%d in decoded body - missing!", i)
		}
	}

	// Expected decoded body: 5 chunks × 43 bytes = 215 bytes
	expectedLength := 5 * 43
	actualLength := len(htResp.Body)

	if actualLength != expectedLength {
		t.Errorf("Expected decoded body length %d bytes, got %d bytes - %d bytes lost!",
			expectedLength, actualLength, expectedLength-actualLength)
	}

	t.Logf("Decoded body length: %d bytes (expected: %d bytes)", actualLength, expectedLength)
	t.Logf("Decoded body:\n%s", decodedBody)
}

// TestChunkedParsingBinaryDataWithNewlineBytes tests that binary data
// containing 0x0A bytes (newline) is handled correctly
func TestChunkedParsingBinaryDataWithNewlineBytes(t *testing.T) {
	// Create binary data with embedded newline bytes
	binaryData1 := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x0A, 0x57, 0x6f, 0x72, 0x6c, 0x64} // "Hello\nWorld" (11 bytes)
	binaryData2 := []byte{0x54, 0x65, 0x73, 0x74, 0x0A, 0x44, 0x61, 0x74, 0x61}             // "Test\nData" (9 bytes)

	// Build chunked response with binary data
	rawResponse := "HTTP/1.1 200 OK\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"Content-Type: application/octet-stream\r\n" +
		"\r\n" +
		"b\r\n" + string(binaryData1) + "\r\n" + // 0xb = 11
		"9\r\n" + string(binaryData2) + "\r\n" + // 0x9 = 9
		"0\r\n" +
		"\r\n"

	opts := response.ParseOptions{
		AutoDecodeChunked: true,
	}

	htResp, err := response.ParseWithOptions([]byte(rawResponse), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Expected decoded body: 11 + 9 = 20 bytes
	expectedLength := 20
	actualLength := len(htResp.Body)

	if actualLength != expectedLength {
		t.Errorf("Expected decoded body length %d bytes, got %d bytes",
			expectedLength, actualLength)
	}

	// Verify the binary data is intact
	expectedBody := append(binaryData1, binaryData2...)
	if string(htResp.Body) != string(expectedBody) {
		t.Error("Binary data was corrupted during chunked parsing")
		t.Logf("Expected: %v", expectedBody)
		t.Logf("Got: %v", htResp.Body)
	}
}
