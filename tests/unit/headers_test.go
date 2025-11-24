package unit

import (
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/headers"
)

func TestOrderedHeaders_Basic(t *testing.T) {
	h := headers.NewOrderedHeaders()

	// Test Set and Get
	h.Set("Content-Type", "application/json")
	h.Set("test", "deneme")

	if got := h.Get("Content-Type"); got != "application/json" {
		t.Errorf("Expected 'application/json', got '%s'", got)
	}

	if got := h.Get("test"); got != "deneme" {
		t.Errorf("Expected 'deneme', got '%s'", got)
	}
}

func TestOrderedHeaders_CaseInsensitive(t *testing.T) {
	h := headers.NewOrderedHeaders()
	h.Set("Content-Type", "application/json")

	// Case insensitive lookup
	if got := h.Get("content-type"); got != "application/json" {
		t.Errorf("Case insensitive lookup failed")
	}

	if got := h.Get("CONTENT-TYPE"); got != "application/json" {
		t.Errorf("Case insensitive lookup failed")
	}
}

func TestOrderedHeaders_OrderPreservation(t *testing.T) {
	h := headers.NewOrderedHeaders()

	// Add headers in specific order
	h.Set("Host", "example.com")
	h.Set("User-Agent", "test")
	h.Set("test", "deneme")
	h.Set("Authorization", "Bearer token")

	all := h.All()
	expected := []string{"Host", "User-Agent", "test", "Authorization"}

	if len(all) != len(expected) {
		t.Errorf("Expected %d headers, got %d", len(expected), len(all))
	}

	for i, header := range all {
		if header.Name != expected[i] {
			t.Errorf("Order mismatch at position %d: expected '%s', got '%s'",
				i, expected[i], header.Name)
		}
	}
}

func TestOrderedHeaders_OriginalCase(t *testing.T) {
	h := headers.NewOrderedHeaders()
	h.Set("Content-Type", "application/json")
	h.Set("X-Custom-Header", "value")

	// Check that original case is preserved
	all := h.All()
	for _, header := range all {
		if header.Name == "content-type" {
			t.Errorf("Original case not preserved: got '%s'", header.Name)
		}
	}

	// Verify GetRaw returns original case
	if got := h.GetRaw("content-type"); got != "Content-Type" {
		t.Errorf("Expected 'Content-Type', got '%s'", got)
	}
}

func TestOrderedHeaders_Update(t *testing.T) {
	h := headers.NewOrderedHeaders()
	h.Set("test", "original")

	// Update existing header
	h.Set("test", "updated")

	if got := h.Get("test"); got != "updated" {
		t.Errorf("Header not updated: got '%s'", got)
	}

	// Should not duplicate in order
	if h.Len() != 1 {
		t.Errorf("Expected 1 header after update, got %d", h.Len())
	}
}

func TestOrderedHeaders_Delete(t *testing.T) {
	h := headers.NewOrderedHeaders()
	h.Set("test", "deneme")
	h.Set("Keep", "this")

	h.Del("test")

	if h.Has("test") {
		t.Error("Header not deleted")
	}

	if !h.Has("Keep") {
		t.Error("Other headers should remain")
	}

	if h.Len() != 1 {
		t.Errorf("Expected 1 header after delete, got %d", h.Len())
	}
}

func TestOrderedHeaders_NonStandard(t *testing.T) {
	h := headers.NewOrderedHeaders()

	// Non-standard header names
	h.Set("test:deneme", "value1")
	h.Set("Weird Header Name", "value2")
	h.Set("123-Numbers", "value3")

	if got := h.Get("test:deneme"); got != "value1" {
		t.Errorf("Non-standard header failed: got '%s'", got)
	}

	if got := h.Get("Weird Header Name"); got != "value2" {
		t.Errorf("Spaced header failed: got '%s'", got)
	}

	if got := h.Get("123-Numbers"); got != "value3" {
		t.Errorf("Numeric header failed: got '%s'", got)
	}
}

func TestOrderedHeaders_Concurrent(t *testing.T) {
	h := headers.NewOrderedHeaders()

	// Test concurrent access (basic smoke test)
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			h.Set("test", "value")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = h.Get("test")
		}
		done <- true
	}()

	<-done
	<-done

	// Should not panic and should have the header
	if !h.Has("test") {
		t.Error("Concurrent access failed")
	}
}

// ==================== FORMAT PRESERVATION TESTS ====================

func TestOrderedHeaders_PreserveOriginalFormat_DoubleSpace(t *testing.T) {
	// Test: "Host:  example.com  " (double space after colon and trailing)
	headerData := []byte("Host:  example.com  \r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Get should return trimmed value
	if got := h.Get("Host"); got != "example.com" {
		t.Errorf("Get() expected 'example.com', got '%s'", got)
	}

	// Build should preserve original format
	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Format not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_PreserveOriginalFormat_NoSpaceAfterColon(t *testing.T) {
	// Test: "X-Custom:value" (no space after colon)
	headerData := []byte("X-Custom:value\r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if got := h.Get("X-Custom"); got != "value" {
		t.Errorf("Get() expected 'value', got '%s'", got)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Format not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_PreserveOriginalFormat_TabAfterColon(t *testing.T) {
	// Test: "X-Tab:\tvalue" (tab after colon)
	headerData := []byte("X-Tab:\tvalue\r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if got := h.Get("X-Tab"); got != "value" {
		t.Errorf("Get() expected 'value', got '%s'", got)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Format not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_PreserveOriginalFormat_MultipleSpaces(t *testing.T) {
	// Test: "X-Spaced:   spaced   " (multiple spaces)
	headerData := []byte("X-Spaced:   spaced   \r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if got := h.Get("X-Spaced"); got != "spaced" {
		t.Errorf("Get() expected 'spaced', got '%s'", got)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Format not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_PreserveLineEnding_LF(t *testing.T) {
	// Test: Line ending with just \n
	headerData := []byte("Host: example.com\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Line ending not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_PreserveLineEnding_CRLF(t *testing.T) {
	// Test: Line ending with \r\n
	headerData := []byte("Host: example.com\r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Line ending not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_PreserveLineEnding_DoubleCR(t *testing.T) {
	// Test: Line ending with \r\r\n (edge case)
	headerData := []byte("Host: example.com\r\r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Line ending not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_PreserveLineEnding_OnlyCR(t *testing.T) {
	// Test: Line ending with just \r (old Mac style)
	headerData := []byte("Host: example.com\rUser-Agent: test\r")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Line ending not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_PreserveMultipleHeaders(t *testing.T) {
	// Test multiple headers with different formats
	headerData := []byte("Host:  example.com  \r\nX-Custom:value\nX-Tab:\tvalue\r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Multiple header format not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_ProgrammaticAdditionUsesStandardFormat(t *testing.T) {
	// Headers added programmatically should use standard format
	h := headers.NewOrderedHeaders()
	h.Set("X-New", "value")

	built := h.Build()
	expected := "X-New: value\r\n"
	if string(built) != expected {
		t.Errorf("Expected standard format %q, got %q", expected, built)
	}
}

func TestOrderedHeaders_SetClearsOriginalFormat(t *testing.T) {
	// When Set() is called on a parsed header, original format should be cleared
	headerData := []byte("Host:  example.com  \r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Update the header programmatically
	h.Set("Host", "newhost.com")

	built := h.Build()
	expected := "Host: newhost.com\r\n"
	if string(built) != expected {
		t.Errorf("Expected standard format after Set(): %q, got %q", expected, built)
	}
}

func TestOrderedHeaders_MixedParsedAndProgrammatic(t *testing.T) {
	// Mix of parsed (preserved) and programmatic (standard) headers
	headerData := []byte("Host:  example.com  \r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Add a new header programmatically
	h.Set("X-New", "value")

	built := h.Build()
	expected := "Host:  example.com  \r\nX-New: value\r\n"
	if string(built) != expected {
		t.Errorf("Mixed format not correct:\nExpected: %q\nGot: %q", expected, built)
	}
}

func TestOrderedHeaders_BuildNormalized(t *testing.T) {
	// BuildNormalized should always use standard format
	headerData := []byte("Host:  example.com  \r\nX-Custom:value\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	built := h.BuildNormalized()
	expected := "Host: example.com\r\nX-Custom: value\r\n"
	if string(built) != expected {
		t.Errorf("BuildNormalized failed:\nExpected: %q\nGot: %q", expected, built)
	}
}

func TestOrderedHeaders_EmptyValue(t *testing.T) {
	// Test header with empty value
	headerData := []byte("X-Empty:\r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if got := h.Get("X-Empty"); got != "" {
		t.Errorf("Get() expected empty string, got '%s'", got)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Empty value header not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}

func TestOrderedHeaders_SpaceInValue(t *testing.T) {
	// Test value with internal spaces (should be preserved)
	headerData := []byte("X-Sentence: Hello World Test\r\n")
	h, err := headers.ParseHeaders(headerData)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if got := h.Get("X-Sentence"); got != "Hello World Test" {
		t.Errorf("Get() expected 'Hello World Test', got '%s'", got)
	}

	built := h.Build()
	if string(built) != string(headerData) {
		t.Errorf("Internal spaces not preserved:\nExpected: %q\nGot: %q", headerData, built)
	}
}
