package unit

import (
	"bytes"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/response"
)

func TestResponseParse_Basic(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Server: nginx/1.18.0
test:deneme

{"message":"success"}`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if resp.Version != "HTTP/1.1" {
		t.Errorf("Expected version HTTP/1.1, got %s", resp.Version)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.StatusText != "OK" {
		t.Errorf("Expected status text OK, got %s", resp.StatusText)
	}

	if got := resp.Headers.Get("test"); got != "deneme" {
		t.Errorf("Expected test header 'deneme', got '%s'", got)
	}

	expectedBody := `{"message":"success"}`
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(resp.Body))
	}
}

func TestResponseParse_FaultTolerance(t *testing.T) {
	// Invalid status code
	raw := []byte(`HTTP/1.1 999 Custom Status
Content-Type: text/plain
test:deneme

Test content`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Fault tolerant parse should succeed: %v", err)
	}

	if resp.StatusCode != 999 {
		t.Errorf("Expected status 999, got %d", resp.StatusCode)
	}

	if resp.StatusText != "Custom Status" {
		t.Errorf("Expected custom status text, got %s", resp.StatusText)
	}
}

func TestResponseParse_DefaultStatusText(t *testing.T) {
	// Missing status text
	raw := []byte(`HTTP/1.1 404
Content-Type: text/html

Not found`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if resp.StatusText != "Not Found" {
		t.Errorf("Expected default status text 'Not Found', got '%s'", resp.StatusText)
	}
}

func TestResponseBuild_Reconstruction(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Server: test-server
test:deneme

{"data":"value"}`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	rebuilt := resp.Build()

	// Parse rebuilt response
	resp2, err := response.Parse(rebuilt)
	if err != nil {
		t.Fatalf("Rebuild parse failed: %v", err)
	}

	// Should be identical
	if resp.StatusCode != resp2.StatusCode {
		t.Errorf("Status code mismatch after rebuild")
	}

	if resp.StatusText != resp2.StatusText {
		t.Errorf("Status text mismatch after rebuild")
	}

	if !bytes.Equal(resp.Body, resp2.Body) {
		t.Errorf("Body mismatch after rebuild")
	}
}

func TestResponseStatusMethods(t *testing.T) {
	tests := []struct {
		status     int
		successful bool
		redirect   bool
		clientErr  bool
		serverErr  bool
	}{
		{200, true, false, false, false},
		{201, true, false, false, false},
		{301, false, true, false, false},
		{404, false, false, true, false},
		{500, false, false, false, true},
	}

	for _, test := range tests {
		resp := response.NewResponse()
		resp.StatusCode = test.status

		if resp.IsSuccessful() != test.successful {
			t.Errorf("Status %d: IsSuccessful() = %t, expected %t",
				test.status, resp.IsSuccessful(), test.successful)
		}

		if resp.IsRedirect() != test.redirect {
			t.Errorf("Status %d: IsRedirect() = %t, expected %t",
				test.status, resp.IsRedirect(), test.redirect)
		}

		if resp.IsClientError() != test.clientErr {
			t.Errorf("Status %d: IsClientError() = %t, expected %t",
				test.status, resp.IsClientError(), test.clientErr)
		}

		if resp.IsServerError() != test.serverErr {
			t.Errorf("Status %d: IsServerError() = %t, expected %t",
				test.status, resp.IsServerError(), test.serverErr)
		}
	}
}

func TestResponseClone(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
test:deneme

{"original":"data"}`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	clone := resp.Clone()

	// Modify original
	resp.StatusCode = 404
	resp.Headers.Set("New-Header", "value")
	resp.Body = []byte("modified")

	// Clone should be unchanged
	if clone.StatusCode != 200 {
		t.Errorf("Clone modified when original changed")
	}

	if clone.Headers.Has("New-Header") {
		t.Errorf("Clone headers modified when original changed")
	}

	if string(clone.Body) == "modified" {
		t.Errorf("Clone body modified when original changed")
	}
}

func TestResponseUtilityMethods(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 15
Server: TestServer
test:deneme

{"test":"data"}`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if resp.GetContentType() != "application/json" {
		t.Errorf("GetContentType failed")
	}

	if resp.GetContentLength() != 15 {
		t.Errorf("GetContentLength failed: got %d", resp.GetContentLength())
	}

	if resp.GetServer() != "TestServer" {
		t.Errorf("GetServer failed")
	}
}

func TestResponseRedirection(t *testing.T) {
	raw := []byte(`HTTP/1.1 302 Found
Location: https://example.com/new-path
test:deneme

`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !resp.IsRedirect() {
		t.Errorf("Should be redirect")
	}

	location := resp.GetRedirectLocation()
	if location != "https://example.com/new-path" {
		t.Errorf("Expected redirect location, got '%s'", location)
	}
}
