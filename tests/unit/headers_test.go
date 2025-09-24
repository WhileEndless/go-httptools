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
