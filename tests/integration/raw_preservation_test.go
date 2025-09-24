package integration

import (
	"bytes"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/request"
)

func TestRawPreservation_ExactReconstruction(t *testing.T) {
	// Test with exact formatting preservation
	original := []byte(`POST    /api/test?param=value   HTTP/1.1
Host:   example.com  
User-Agent:Mozilla/5.0    
test:deneme
Content-Type:  application/json  
Authorization:   Bearer   token123  

{  "data":  "test"  }`)

	rawReq, err := request.ParseRaw(original)
	if err != nil {
		t.Fatalf("ParseRaw failed: %v", err)
	}

	// Verify parsed correctly despite formatting issues
	if rawReq.Method != "POST" {
		t.Errorf("Method parsing failed: got %s", rawReq.Method)
	}

	if rawReq.URL != "/api/test?param=value" {
		t.Errorf("URL parsing failed: got %s", rawReq.URL)
	}

	// Check that custom header is preserved
	if rawReq.Headers.Get("test") != "deneme" {
		t.Errorf("Custom header parsing failed")
	}

	// Rebuild should be identical to original
	rebuilt := rawReq.BuildRaw()

	if !bytes.Equal(original, rebuilt) {
		t.Errorf("Raw reconstruction not identical:\nOriginal:\n%q\nRebuilt:\n%q",
			string(original), string(rebuilt))
	}
}

func TestRawPreservation_WeirdSpacing(t *testing.T) {
	// Request with weird spacing that should be preserved exactly
	original := []byte(`GET     /path/with/spaces     HTTP/1.1
Host:example.com
test:deneme
User-Agent:  TestAgent  
Content-Type:application/json  

`)

	rawReq, err := request.ParseRaw(original)
	if err != nil {
		t.Fatalf("ParseRaw failed: %v", err)
	}

	rebuilt := rawReq.BuildRaw()

	if !bytes.Equal(original, rebuilt) {
		t.Errorf("Raw spacing not preserved:\nOriginal:\n%q\nRebuilt:\n%q",
			string(original), string(rebuilt))
	}

	// Verify custom header still accessible
	if rawReq.Headers.Get("test") != "deneme" {
		t.Errorf("Custom header lost in weird spacing")
	}
}

func TestRawPreservation_NonStandardHeaders(t *testing.T) {
	// Various non-standard headers with exact spacing
	original := []byte(`PATCH /api HTTP/1.1
Host: example.com
test:deneme
X-123-Numbers: numeric header
Weird Header Name: spaces in name
header-without-value:
UPPERCASE-HEADER: value

{"patch": "data"}`)

	rawReq, err := request.ParseRaw(original)
	if err != nil {
		t.Fatalf("ParseRaw failed: %v", err)
	}

	rebuilt := rawReq.BuildRaw()

	if !bytes.Equal(original, rebuilt) {
		t.Errorf("Raw non-standard headers not preserved:\nOriginal:\n%q\nRebuilt:\n%q",
			string(original), string(rebuilt))
	}

	// Verify all headers are accessible
	if rawReq.Headers.Get("test") != "deneme" {
		t.Errorf("test header lost")
	}

	if rawReq.Headers.Get("X-123-Numbers") != "numeric header" {
		t.Errorf("Numeric header lost")
	}

	if rawReq.Headers.Get("Weird Header Name") != "spaces in name" {
		t.Errorf("Spaced header name lost")
	}

	// Empty value header should be preserved
	if !rawReq.Headers.Has("header-without-value") {
		t.Errorf("Empty value header lost")
	}
}

func TestRawPreservation_StandardCompatibility(t *testing.T) {
	// Test conversion between raw and standard formats
	original := []byte(`POST /api/test HTTP/1.1
Host: example.com
test:deneme
Content-Type: application/json

{"data": "test"}`)

	// Parse as raw
	rawReq, err := request.ParseRaw(original)
	if err != nil {
		t.Fatalf("ParseRaw failed: %v", err)
	}

	// Convert to standard
	stdReq := rawReq.ToStandard()

	// Should have same data
	if stdReq.Method != rawReq.Method {
		t.Errorf("Method conversion failed")
	}

	if stdReq.URL != rawReq.URL {
		t.Errorf("URL conversion failed")
	}

	if stdReq.Headers.Get("test") != "deneme" {
		t.Errorf("Custom header lost in conversion")
	}

	if !bytes.Equal(stdReq.Body, rawReq.Body) {
		t.Errorf("Body conversion failed")
	}

	// Convert back to raw (will lose original formatting)
	rawReq2 := request.FromStandard(stdReq)

	// Should have same parsed data
	if rawReq2.Method != rawReq.Method {
		t.Errorf("Round-trip method failed")
	}

	if rawReq2.Headers.Get("test") != "deneme" {
		t.Errorf("Round-trip custom header failed")
	}
}

func TestRawPreservation_EditingMaintainsFormat(t *testing.T) {
	// Test that editing one header preserves formatting of others
	original := []byte(`POST   /api/users   HTTP/1.1
Host:  example.com  
test:deneme
Authorization:  Bearer old-token  
Content-Type:  application/json  

{"original": "data"}`)

	rawReq, err := request.ParseRaw(original)
	if err != nil {
		t.Fatalf("ParseRaw failed: %v", err)
	}

	// Edit only Authorization header
	rawReq.Headers.Set("Authorization", "Bearer new-token")

	rebuilt := rawReq.BuildRaw()

	// Parse both to compare
	originalReq, _ := request.ParseRaw(original)
	modifiedReq, _ := request.ParseRaw(rebuilt)

	// Everything except Authorization should be functionally identical
	if originalReq.Method != modifiedReq.Method {
		t.Errorf("Method changed unexpectedly")
	}

	if originalReq.URL != modifiedReq.URL {
		t.Errorf("URL changed unexpectedly")
	}

	if !bytes.Equal(originalReq.Body, modifiedReq.Body) {
		t.Errorf("Body changed unexpectedly")
	}

	// Custom header should still be there
	if modifiedReq.Headers.Get("test") != "deneme" {
		t.Errorf("Custom header lost during editing")
	}

	// Authorization should be updated
	if modifiedReq.Headers.Get("Authorization") != "Bearer new-token" {
		t.Errorf("Authorization not updated correctly")
	}

	// Other headers should maintain their values
	if modifiedReq.Headers.Get("Host") != "example.com" {
		t.Errorf("Host header changed unexpectedly")
	}
}
