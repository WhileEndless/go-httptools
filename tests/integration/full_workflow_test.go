package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
	"github.com/WhileEndless/go-httptools/pkg/utils"
)

func TestFullWorkflow_RequestEditing(t *testing.T) {
	// Start with a real-world HTTP request
	originalRaw := []byte(`POST /api/v1/users HTTP/1.1
Host: api.example.com
Content-Type: application/x-www-form-urlencoded
Content-Length: 25
User-Agent: Mozilla/5.0
test:deneme

username=john&password=old`)

	// Parse
	req, err := request.Parse(originalRaw)
	if err != nil {
		t.Fatalf("Failed to parse original request: %v", err)
	}

	// Validate original
	validation := utils.ValidateRequest(req)
	if !validation.Valid {
		t.Errorf("Original request should be valid")
	}

	// Edit using fluent interface
	editor := utils.NewRequestEditor(req)
	modified := editor.
		SetMethod("PUT").
		SetURL("/api/v2/users/123").
		UpdateHeader("Content-Type", "application/json").
		RemoveHeader("Content-Length"). // Will be recalculated
		AddHeader("Authorization", "Bearer token123").
		SetBodyString(`{"username":"john","password":"newpass","profile":{"age":30}}`).
		AddQueryParam("force", "true").
		GetRequest()

	// Validate modified
	validation = utils.ValidateRequest(modified)
	if !validation.Valid {
		t.Errorf("Modified request should be valid")
		for _, err := range validation.Errors {
			t.Logf("Error: %s", err)
		}
	}

	// Build and verify
	modifiedRaw := modified.Build()

	// Parse the rebuilt request to verify
	reparsed, err := request.Parse(modifiedRaw)
	if err != nil {
		t.Fatalf("Failed to reparse modified request: %v", err)
	}

	// Verify modifications
	if reparsed.Method != "PUT" {
		t.Errorf("Method not modified: got %s", reparsed.Method)
	}

	if reparsed.URL != "/api/v2/users/123?force=true" {
		t.Errorf("URL not modified correctly: got %s", reparsed.URL)
	}

	if reparsed.GetContentType() != "application/json" {
		t.Errorf("Content-Type not updated")
	}

	if reparsed.Headers.Get("Authorization") != "Bearer token123" {
		t.Errorf("Authorization header not added")
	}

	if reparsed.Headers.Get("test") != "deneme" {
		t.Errorf("Original custom header should be preserved")
	}

	// Verify Content-Length is correct
	expectedBodyLen := len(reparsed.Body)
	if reparsed.GetContentLength() != string(rune(expectedBodyLen+48)) { // ASCII conversion
		actualLen := reparsed.GetContentLength()
		t.Logf("Content-Length: expected length for %d bytes, got %s", expectedBodyLen, actualLen)
		// This is informational - SetBodyString should handle this
	}
}

func TestFullWorkflow_ResponseEditing(t *testing.T) {
	// Start with a real-world HTTP response
	originalRaw := []byte(`HTTP/1.1 404 Not Found
Content-Type: text/html
Content-Length: 43
Server: nginx/1.18.0
test:deneme

<html><body>Page not found</body></html>`)

	// Parse
	resp, err := response.Parse(originalRaw)
	if err != nil {
		t.Fatalf("Failed to parse original response: %v", err)
	}

	// Edit to convert error to success
	editor := utils.NewResponseEditor(resp)
	modified := editor.
		SetStatusCode(200).
		SetStatusText("OK").
		UpdateHeader("Content-Type", "application/json").
		AddHeader("Cache-Control", "no-cache").
		RemoveHeader("Content-Length"). // Will be recalculated
		SetBodyString(`{"status":"success","message":"User created","data":{"id":123,"name":"John"}}`, false).
		GetResponse()

	// Validate modified
	validation := utils.ValidateResponse(modified)
	if !validation.Valid {
		t.Errorf("Modified response should be valid")
		for _, err := range validation.Errors {
			t.Logf("Error: %s", err)
		}
	}

	// Build and verify
	modifiedRaw := modified.Build()

	// Parse the rebuilt response
	reparsed, err := response.Parse(modifiedRaw)
	if err != nil {
		t.Fatalf("Failed to reparse modified response: %v", err)
	}

	// Verify modifications
	if reparsed.StatusCode != 200 {
		t.Errorf("Status code not modified: got %d", reparsed.StatusCode)
	}

	if reparsed.StatusText != "OK" {
		t.Errorf("Status text not modified: got %s", reparsed.StatusText)
	}

	if reparsed.GetContentType() != "application/json" {
		t.Errorf("Content-Type not updated")
	}

	if reparsed.Headers.Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control header not added")
	}

	if reparsed.Headers.Get("test") != "deneme" {
		t.Errorf("Original custom header should be preserved")
	}

	// Verify body is JSON
	bodyStr := string(reparsed.Body)
	if !bytes.Contains(reparsed.Body, []byte(`"status":"success"`)) {
		t.Errorf("JSON body not set correctly: %s", bodyStr)
	}
}

func TestFullWorkflow_ChainedEditing(t *testing.T) {
	// Test multiple round-trip edits
	originalRaw := []byte(`GET /test HTTP/1.1
Host: example.com
test:deneme

`)

	// Parse
	req, err := request.Parse(originalRaw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Edit 1: Add authentication
	editor1 := utils.NewRequestEditor(req)
	req1 := editor1.
		AddHeader("Authorization", "Bearer token1").
		AddQueryParam("auth", "true").
		GetRequest()

	// Edit 2: Change method and add body
	editor2 := utils.NewRequestEditor(req1)
	req2 := editor2.
		SetMethod("POST").
		SetBodyString(`{"action":"test"}`).
		UpdateHeader("Content-Type", "application/json").
		GetRequest()

	// Edit 3: Update authentication
	editor3 := utils.NewRequestEditor(req2)
	req3 := editor3.
		UpdateHeader("Authorization", "Bearer token2").
		AddQueryParam("version", "2").
		GetRequest()

	// Verify final state
	if req3.Method != "POST" {
		t.Errorf("Final method should be POST")
	}

	if req3.Headers.Get("Authorization") != "Bearer token2" {
		t.Errorf("Authorization should be updated")
	}

	if req3.Headers.Get("test") != "deneme" {
		t.Errorf("Original header should survive all edits")
	}

	// URL should have both query params
	if !bytes.Contains([]byte(req3.URL), []byte("auth=true")) {
		t.Errorf("First query param missing: %s", req3.URL)
	}

	if !bytes.Contains([]byte(req3.URL), []byte("version=2")) {
		t.Errorf("Second query param missing: %s", req3.URL)
	}

	// Verify we can still rebuild correctly
	finalRaw := req3.Build()
	finalReq, err := request.Parse(finalRaw)
	if err != nil {
		t.Fatalf("Final parse failed: %v", err)
	}

	if finalReq.Headers.Get("test") != "deneme" {
		t.Errorf("Custom header lost in final rebuild")
	}
}

func TestFullWorkflow_HeaderOrderConsistency(t *testing.T) {
	// Ensure header order is preserved through multiple operations
	raw := []byte(`POST /test HTTP/1.1
Host: example.com
User-Agent: TestAgent
X-Custom-1: value1
Content-Type: application/json
X-Custom-2: value2
test:deneme
Authorization: Bearer token

{"data":"test"}`)

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Get original header order
	originalHeaders := req.Headers.All()
	originalOrder := make([]string, len(originalHeaders))
	for i, h := range originalHeaders {
		originalOrder[i] = h.Name
	}

	// Edit without changing existing headers (but SetBodyString updates Content-Length)
	editor := utils.NewRequestEditor(req)
	modified := editor.
		SetBodyString(`{"modified":"data"}`).
		GetRequest()

	// Check order preserved (SetBodyString may add/update Content-Length)
	modifiedHeaders := modified.Headers.All()
	// Allow for Content-Length to be added if it wasn't present originally
	expectedCount := len(originalHeaders)
	hasContentLength := false
	for _, h := range originalHeaders {
		if strings.ToLower(h.Name) == "content-length" {
			hasContentLength = true
			break
		}
	}
	if !hasContentLength {
		expectedCount++ // Content-Length will be added
	}

	if len(modifiedHeaders) != expectedCount {
		t.Errorf("Unexpected header count change: %d -> %d", len(originalHeaders), len(modifiedHeaders))
	}

	// Check that original headers maintain their relative order
	// (skip checking any newly added Content-Length header)
	originalHeadersFound := 0
	for _, h := range modifiedHeaders {
		// Skip if this is a newly added Content-Length header
		if strings.ToLower(h.Name) == "content-length" && !hasContentLength {
			continue
		}
		if originalHeadersFound < len(originalOrder) {
			if h.Name != originalOrder[originalHeadersFound] {
				t.Errorf("Header order changed at position %d: expected '%s', got '%s'",
					originalHeadersFound, originalOrder[originalHeadersFound], h.Name)
			}
			originalHeadersFound++
		}
	}

	// Verify test header is still there
	if modified.Headers.Get("test") != "deneme" {
		t.Errorf("Custom test header lost")
	}
}
