package unit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/request"
)

// TestBodyParsing_64KB_Boundary tests the exact 64KB boundary
// This test verifies the fix for the bufio.Scanner 64KB limitation bug
func TestBodyParsing_64KB_Boundary(t *testing.T) {
	tests := []struct {
		name     string
		bodySize int
	}{
		{
			name:     "Small body (1KB)",
			bodySize: 1024,
		},
		{
			name:     "Medium body (32KB)",
			bodySize: 32 * 1024,
		},
		{
			name:     "Exact 64KB boundary",
			bodySize: 64 * 1024,
		},
		{
			name:     "64KB + 1 byte",
			bodySize: 64*1024 + 1,
		},
		{
			name:     "128KB",
			bodySize: 128 * 1024,
		},
		{
			name:     "1MB",
			bodySize: 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create body of specified size
			body := strings.Repeat("A", tt.bodySize)

			// Build raw HTTP request
			rawRequest := fmt.Sprintf(
				"POST /api/upload HTTP/1.1\r\n"+
					"Host: example.com\r\n"+
					"Content-Type: text/plain\r\n"+
					"Content-Length: %d\r\n"+
					"\r\n"+
					"%s",
				len(body),
				body,
			)

			// Parse request
			req, err := request.Parse([]byte(rawRequest))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Verify Raw contains full request
			if len(req.Raw) != len(rawRequest) {
				t.Errorf("Raw length mismatch: got %d, want %d", len(req.Raw), len(rawRequest))
			}

			// Verify Body is parsed correctly
			if len(req.Body) != len(body) {
				t.Errorf("Body length mismatch: got %d, want %d", len(req.Body), len(body))
			}

			if len(req.Body) > 0 && string(req.Body) != body {
				t.Errorf("Body content mismatch")
			}
		})
	}
}

// TestBodyParsing_LargeJSON tests parsing large JSON payloads
func TestBodyParsing_LargeJSON(t *testing.T) {
	// Create a large JSON payload (100KB)
	jsonItems := make([]string, 1000)
	for i := range jsonItems {
		jsonItems[i] = fmt.Sprintf(`{"id":%d,"name":"item_%d","data":"%s"}`,
			i, i, strings.Repeat("x", 50))
	}
	body := "[" + strings.Join(jsonItems, ",") + "]"

	rawRequest := fmt.Sprintf(
		"POST /api/bulk HTTP/1.1\r\n"+
			"Host: api.example.com\r\n"+
			"Content-Type: application/json\r\n"+
			"Content-Length: %d\r\n"+
			"\r\n"+
			"%s",
		len(body),
		body,
	)

	req, err := request.Parse([]byte(rawRequest))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(req.Body) != len(body) {
		t.Errorf("JSON body length mismatch: got %d, want %d", len(req.Body), len(body))
	}

	if string(req.Body) != body {
		t.Errorf("JSON body content mismatch")
	}
}

// TestBodyParsing_BinaryData tests parsing binary request bodies
func TestBodyParsing_BinaryData(t *testing.T) {
	// Create binary data (200KB)
	body := make([]byte, 200*1024)
	for i := range body {
		body[i] = byte(i % 256)
	}

	// Build request with binary body
	headers := fmt.Sprintf(
		"POST /api/upload HTTP/1.1\r\n"+
			"Host: files.example.com\r\n"+
			"Content-Type: application/octet-stream\r\n"+
			"Content-Length: %d\r\n"+
			"\r\n",
		len(body),
	)

	rawRequest := append([]byte(headers), body...)

	req, err := request.Parse(rawRequest)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(req.Body) != len(body) {
		t.Errorf("Binary body length mismatch: got %d, want %d", len(req.Body), len(body))
	}

	// Verify binary content matches
	if len(req.Body) > 0 {
		for i := 0; i < len(body) && i < len(req.Body); i++ {
			if req.Body[i] != body[i] {
				t.Errorf("Binary data mismatch at byte %d: got %d, want %d", i, req.Body[i], body[i])
				break
			}
		}
	}
}

// TestBodyParsing_WithNewlines tests bodies containing newlines
func TestBodyParsing_WithNewlines(t *testing.T) {
	// Body with lots of newlines (100KB total)
	lines := make([]string, 2000)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d with some data %s", i, strings.Repeat("x", 30))
	}
	body := strings.Join(lines, "\n")

	rawRequest := fmt.Sprintf(
		"POST /api/lines HTTP/1.1\r\n"+
			"Host: example.com\r\n"+
			"Content-Type: text/plain\r\n"+
			"Content-Length: %d\r\n"+
			"\r\n"+
			"%s",
		len(body),
		body,
	)

	req, err := request.Parse([]byte(rawRequest))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(req.Body) != len(body) {
		t.Errorf("Body with newlines length mismatch: got %d, want %d", len(req.Body), len(body))
	}

	// Verify newlines are preserved
	if len(req.Body) > 0 && string(req.Body) != body {
		t.Errorf("Body content with newlines doesn't match")
	}
}

// TestBodyParsing_MultipleRequests tests multiple requests with varying sizes
func TestBodyParsing_MultipleRequests(t *testing.T) {
	sizes := []int{
		1024,            // 1KB
		32 * 1024,       // 32KB
		64 * 1024,       // 64KB (boundary)
		64*1024 + 1,     // 64KB + 1
		128 * 1024,      // 128KB
		512 * 1024,      // 512KB
		1024 * 1024,     // 1MB
		5 * 1024 * 1024, // 5MB
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%dKB", size/1024), func(t *testing.T) {
			body := strings.Repeat("X", size)

			rawRequest := fmt.Sprintf(
				"PUT /api/data HTTP/1.1\r\n"+
					"Host: example.com\r\n"+
					"Content-Length: %d\r\n"+
					"\r\n"+
					"%s",
				len(body),
				body,
			)

			req, err := request.Parse([]byte(rawRequest))
			if err != nil {
				t.Fatalf("Parse failed for size %d: %v", size, err)
			}

			if len(req.Body) != len(body) {
				t.Errorf("Body length mismatch: got %d, want %d", len(req.Body), len(body))
			}
		})
	}
}

// TestBodyParsing_EmptyBody tests that empty bodies are handled correctly
func TestBodyParsing_EmptyBody(t *testing.T) {
	rawRequest := "GET /api/test HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	req, err := request.Parse([]byte(rawRequest))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(req.Body) != 0 {
		t.Errorf("Empty body should have length 0, got %d", len(req.Body))
	}
}

// TestBodyParsing_BodyWithCRLF tests that CRLF in body is preserved
func TestBodyParsing_BodyWithCRLF(t *testing.T) {
	body := "Line1\r\nLine2\r\nLine3"

	rawRequest := fmt.Sprintf(
		"POST /api/test HTTP/1.1\r\n"+
			"Host: example.com\r\n"+
			"Content-Length: %d\r\n"+
			"\r\n"+
			"%s",
		len(body),
		body,
	)

	req, err := request.Parse([]byte(rawRequest))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if string(req.Body) != body {
		t.Errorf("Body with CRLF mismatch: got %q, want %q", string(req.Body), body)
	}
}

// BenchmarkBodyParsing benchmarks parsing with different body sizes
func BenchmarkBodyParsing(b *testing.B) {
	sizes := []int{
		1024,        // 1KB
		64 * 1024,   // 64KB
		128 * 1024,  // 128KB
		1024 * 1024, // 1MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			body := strings.Repeat("B", size)
			rawRequest := fmt.Sprintf(
				"POST /benchmark HTTP/1.1\r\n"+
					"Host: bench.example.com\r\n"+
					"Content-Length: %d\r\n"+
					"\r\n"+
					"%s",
				len(body),
				body,
			)

			rawBytes := []byte(rawRequest)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := request.Parse(rawBytes)
				if err != nil {
					b.Fatalf("Parse failed: %v", err)
				}
			}
		})
	}
}
