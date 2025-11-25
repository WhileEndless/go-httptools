package unit

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/http2"
	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
)

// ==================== HEADER LIST TESTS ====================

func TestHeaderList_BasicOperations(t *testing.T) {
	h := http2.NewHeaderList()

	h.Add("content-type", "application/json")
	h.Add("x-custom", "value1")

	if h.Len() != 2 {
		t.Errorf("Expected 2 headers, got %d", h.Len())
	}

	if got := h.Get("content-type"); got != "application/json" {
		t.Errorf("Expected 'application/json', got '%s'", got)
	}
}

func TestHeaderList_OrderPreservation(t *testing.T) {
	h := http2.NewHeaderList()

	h.Add("first", "1")
	h.Add("second", "2")
	h.Add("third", "3")

	all := h.All()
	if len(all) != 3 {
		t.Fatalf("Expected 3 headers, got %d", len(all))
	}

	expected := []string{"first", "second", "third"}
	for i, hdr := range all {
		if hdr.Name != expected[i] {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expected[i], hdr.Name)
		}
	}
}

func TestHeaderList_InsertAt(t *testing.T) {
	h := http2.NewHeaderList()
	h.Add("first", "1")
	h.Add("third", "3")

	// Insert at position 1
	h.InsertAt(1, "second", "2")

	all := h.All()
	if all[1].Name != "second" {
		t.Errorf("InsertAt failed: expected 'second' at position 1, got '%s'", all[1].Name)
	}
}

func TestHeaderList_InsertBefore(t *testing.T) {
	h := http2.NewHeaderList()
	h.Add("host", "example.com")
	h.Add("content-type", "text/html")

	h.InsertBefore("content-type", "accept", "*/*")

	all := h.All()
	if all[1].Name != "accept" {
		t.Errorf("InsertBefore failed: expected 'accept' at position 1, got '%s'", all[1].Name)
	}
}

func TestHeaderList_InsertAfter(t *testing.T) {
	h := http2.NewHeaderList()
	h.Add("host", "example.com")
	h.Add("content-type", "text/html")

	h.InsertAfter("host", "accept", "*/*")

	all := h.All()
	if all[1].Name != "accept" {
		t.Errorf("InsertAfter failed: expected 'accept' at position 1, got '%s'", all[1].Name)
	}
}

func TestHeaderList_MoveToFront(t *testing.T) {
	h := http2.NewHeaderList()
	h.Add("first", "1")
	h.Add("second", "2")
	h.Add("third", "3")

	h.MoveToFront("third")

	all := h.All()
	if all[0].Name != "third" {
		t.Errorf("MoveToFront failed: expected 'third' at position 0, got '%s'", all[0].Name)
	}
}

func TestHeaderList_MoveToBack(t *testing.T) {
	h := http2.NewHeaderList()
	h.Add("first", "1")
	h.Add("second", "2")
	h.Add("third", "3")

	h.MoveToBack("first")

	all := h.All()
	if all[2].Name != "first" {
		t.Errorf("MoveToBack failed: expected 'first' at position 2, got '%s'", all[2].Name)
	}
}

func TestHeaderList_Clone(t *testing.T) {
	h := http2.NewHeaderList()
	h.Add("test", "value")

	clone := h.Clone()
	clone.Set("test", "modified")

	if h.Get("test") == "modified" {
		t.Error("Clone is not independent: original was modified")
	}
}

func TestHeaderList_CaseInsensitive(t *testing.T) {
	h := http2.NewHeaderList()
	h.Add("Content-Type", "application/json")

	if got := h.Get("content-type"); got != "application/json" {
		t.Errorf("Case insensitive lookup failed: got '%s'", got)
	}

	if got := h.Get("CONTENT-TYPE"); got != "application/json" {
		t.Errorf("Case insensitive lookup failed: got '%s'", got)
	}
}

// ==================== HTTP/2 REQUEST TESTS ====================

func TestHTTP2Request_Basic(t *testing.T) {
	req := http2.NewRequest()
	req.Method = "POST"
	req.Scheme = "https"
	req.Authority = "api.example.com"
	req.Path = "/v1/users"
	req.Headers.Add("content-type", "application/json")
	req.Body = []byte(`{"name":"test"}`)

	if req.Method != "POST" {
		t.Errorf("Method not set correctly")
	}

	all := req.GetAllHeaders()
	if len(all) != 5 { // 4 pseudo-headers + 1 regular header
		t.Errorf("Expected 5 headers, got %d", len(all))
	}

	// Check pseudo-headers come first
	if all[0].Name != ":method" {
		t.Errorf("First header should be :method, got %s", all[0].Name)
	}
}

func TestHTTP2Request_Clone(t *testing.T) {
	req := http2.NewRequest()
	req.Method = "GET"
	req.Authority = "example.com"
	req.Headers.Add("x-custom", "value")

	clone := req.Clone()
	clone.Method = "POST"
	clone.Headers.Set("x-custom", "modified")

	if req.Method != "GET" {
		t.Error("Clone modified original request method")
	}

	if req.Headers.Get("x-custom") != "value" {
		t.Error("Clone modified original request headers")
	}
}

func TestHTTP2Request_JSON(t *testing.T) {
	req := http2.NewRequest()
	req.Method = "POST"
	req.Authority = "api.example.com"
	req.Path = "/test"
	req.Headers.Add("content-type", "application/json")

	jsonBytes, err := req.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Parse back
	req2 := http2.NewRequest()
	err = req2.FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if req2.Method != req.Method {
		t.Errorf("Method mismatch after JSON round-trip")
	}

	if req2.Authority != req.Authority {
		t.Errorf("Authority mismatch after JSON round-trip")
	}
}

func TestHTTP2Request_Build(t *testing.T) {
	req := http2.NewRequest()
	req.Method = "GET"
	req.Scheme = "https"
	req.Authority = "example.com"
	req.Path = "/api"
	req.Headers.Add("accept", "*/*")

	built := req.Build()
	builtStr := string(built)

	if !strings.Contains(builtStr, ":method: GET") {
		t.Error("Build should contain :method pseudo-header")
	}

	if !strings.Contains(builtStr, ":authority: example.com") {
		t.Error("Build should contain :authority pseudo-header")
	}

	if !strings.Contains(builtStr, "accept: */*") {
		t.Error("Build should contain accept header")
	}
}

func TestHTTP2Request_BuildAsHTTP1(t *testing.T) {
	req := http2.NewRequest()
	req.Method = "GET"
	req.Authority = "example.com"
	req.Path = "/api/test"
	req.Headers.Add("accept", "application/json")

	built := req.BuildAsHTTP1()
	builtStr := string(built)

	if !strings.HasPrefix(builtStr, "GET /api/test HTTP/1.1") {
		t.Errorf("BuildAsHTTP1 should start with HTTP/1.1 request line, got: %s", builtStr[:50])
	}

	if !strings.Contains(builtStr, "Host: example.com") {
		t.Error("BuildAsHTTP1 should contain Host header from :authority")
	}
}

// ==================== HTTP/2 RESPONSE TESTS ====================

func TestHTTP2Response_Basic(t *testing.T) {
	resp := http2.NewResponse()
	resp.Status = 200
	resp.Headers.Add("content-type", "application/json")
	resp.Body = []byte(`{"status":"ok"}`)

	if resp.Status != 200 {
		t.Errorf("Status not set correctly")
	}

	if resp.GetStatusText() != "OK" {
		t.Errorf("GetStatusText failed: got '%s'", resp.GetStatusText())
	}
}

func TestHTTP2Response_GetAllHeaders(t *testing.T) {
	resp := http2.NewResponse()
	resp.Status = 404
	resp.Headers.Add("content-type", "text/html")

	all := resp.GetAllHeaders()
	if len(all) != 2 { // 1 pseudo-header + 1 regular header
		t.Errorf("Expected 2 headers, got %d", len(all))
	}

	if all[0].Name != ":status" {
		t.Errorf("First header should be :status, got %s", all[0].Name)
	}

	if all[0].Value != "404" {
		t.Errorf(":status value should be '404', got '%s'", all[0].Value)
	}
}

func TestHTTP2Response_BuildAsHTTP1(t *testing.T) {
	resp := http2.NewResponse()
	resp.Status = 200
	resp.Headers.Add("content-type", "application/json")
	resp.Body = []byte(`{"ok":true}`)

	built := resp.BuildAsHTTP1()
	builtStr := string(built)

	if !strings.HasPrefix(builtStr, "HTTP/1.1 200 OK") {
		t.Errorf("BuildAsHTTP1 should start with HTTP/1.1 status line, got: %s", builtStr[:30])
	}

	if !strings.Contains(builtStr, "Content-Length: 11") {
		t.Error("BuildAsHTTP1 should add Content-Length header")
	}
}

// ==================== CONVERSION TESTS ====================

func TestFromHTTP1Request(t *testing.T) {
	raw := []byte("POST /api/users HTTP/1.1\r\nHost: api.example.com\r\nContent-Type: application/json\r\nAuthorization: Bearer token\r\n\r\n{\"name\":\"test\"}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	h2req := http2.FromHTTP1Request(req)

	if h2req.Method != "POST" {
		t.Errorf("Method not converted: expected 'POST', got '%s'", h2req.Method)
	}

	if h2req.Authority != "api.example.com" {
		t.Errorf("Authority not set from Host: expected 'api.example.com', got '%s'", h2req.Authority)
	}

	if h2req.Path != "/api/users" {
		t.Errorf("Path not converted: expected '/api/users', got '%s'", h2req.Path)
	}

	// Host header should be excluded (it's in :authority)
	if h2req.Headers.Has("host") {
		t.Error("Host header should be excluded from regular headers")
	}

	// Other headers should be present (case insensitive)
	if got := h2req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type header not preserved: got '%s'", got)
	}
}

func TestToHTTP1Request(t *testing.T) {
	h2req := http2.NewRequest()
	h2req.Method = "GET"
	h2req.Scheme = "https"
	h2req.Authority = "example.com"
	h2req.Path = "/api/test"
	h2req.Headers.Add("accept", "application/json")

	req := http2.ToHTTP1Request(h2req)

	if req.Method != "GET" {
		t.Errorf("Method not converted: got '%s'", req.Method)
	}

	if req.URL != "/api/test" {
		t.Errorf("URL not converted: got '%s'", req.URL)
	}

	if strings.TrimSpace(req.Headers.Get("Host")) != "example.com" {
		t.Errorf("Host header not set from :authority: got '%s'", req.Headers.Get("Host"))
	}
}

func TestFromHTTP1Response(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nServer: nginx\r\n\r\n{\"status\":\"ok\"}")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	h2resp := http2.FromHTTP1Response(resp)

	if h2resp.Status != 200 {
		t.Errorf("Status not converted: expected 200, got %d", h2resp.Status)
	}

	if got := h2resp.Headers.Get("Server"); got != "nginx" {
		t.Errorf("Server header not preserved: got '%s'", got)
	}
}

func TestToHTTP1Response(t *testing.T) {
	h2resp := http2.NewResponse()
	h2resp.Status = 404
	h2resp.Headers.Add("content-type", "text/html")
	h2resp.Body = []byte("<h1>Not Found</h1>")

	resp := http2.ToHTTP1Response(h2resp)

	if resp.StatusCode != 404 {
		t.Errorf("StatusCode not converted: got %d", resp.StatusCode)
	}

	if resp.StatusText != "Not Found" {
		t.Errorf("StatusText not set: got '%s'", resp.StatusText)
	}
}

func TestParseRequestHeaders(t *testing.T) {
	fields := []http2.HeaderField{
		{Name: ":method", Value: "POST"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "api.example.com"},
		{Name: ":path", Value: "/users"},
		{Name: "content-type", Value: "application/json"},
	}

	req := http2.ParseRequestHeaders(fields)

	if req.Method != "POST" {
		t.Errorf("Method not parsed: got '%s'", req.Method)
	}

	if req.Scheme != "https" {
		t.Errorf("Scheme not parsed: got '%s'", req.Scheme)
	}

	if req.Authority != "api.example.com" {
		t.Errorf("Authority not parsed: got '%s'", req.Authority)
	}

	if req.Path != "/users" {
		t.Errorf("Path not parsed: got '%s'", req.Path)
	}

	if req.Headers.Get("content-type") != "application/json" {
		t.Error("Regular header not parsed")
	}
}

func TestParseResponseHeaders(t *testing.T) {
	fields := []http2.HeaderField{
		{Name: ":status", Value: "201"},
		{Name: "content-type", Value: "application/json"},
		{Name: "location", Value: "/users/123"},
	}

	resp := http2.ParseResponseHeaders(fields)

	if resp.Status != 201 {
		t.Errorf("Status not parsed: got %d", resp.Status)
	}

	if resp.Headers.Get("location") != "/users/123" {
		t.Error("Location header not parsed")
	}
}

// ==================== HEADER ORDER PRESERVATION TESTS ====================

func TestHTTP2_HeaderOrderPreserved_AfterConversion(t *testing.T) {
	// Create HTTP/1.1 request with specific header order
	raw := []byte("GET /api HTTP/1.1\r\nHost: example.com\r\nAccept: */*\r\nAuthorization: Bearer token\r\nContent-Type: application/json\r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Convert to HTTP/2
	h2req := http2.FromHTTP1Request(req)

	// Check header order is preserved (excluding Host which goes to :authority)
	expectedOrder := []string{"accept", "authorization", "content-type"}
	allHeaders := h2req.Headers.All()

	if len(allHeaders) != len(expectedOrder) {
		t.Fatalf("Expected %d headers, got %d", len(expectedOrder), len(allHeaders))
	}

	for i, expected := range expectedOrder {
		if strings.ToLower(allHeaders[i].Name) != expected {
			t.Errorf("Header order not preserved at position %d: expected '%s', got '%s'",
				i, expected, allHeaders[i].Name)
		}
	}
}

func TestHTTP2_HeaderOrderPreserved_JSON_RoundTrip(t *testing.T) {
	req := http2.NewRequest()
	req.Method = "GET"
	req.Path = "/"
	req.Headers.Add("first", "1")
	req.Headers.Add("second", "2")
	req.Headers.Add("third", "3")

	// To JSON
	jsonBytes, err := req.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// From JSON
	req2 := http2.NewRequest()
	err = req2.FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// Check order
	all1 := req.Headers.All()
	all2 := req2.Headers.All()

	if len(all1) != len(all2) {
		t.Fatalf("Header count mismatch after JSON round-trip")
	}

	for i := range all1 {
		if all1[i].Name != all2[i].Name {
			t.Errorf("Header order not preserved at position %d: expected '%s', got '%s'",
				i, all1[i].Name, all2[i].Name)
		}
	}
}

// ==================== JSON SERIALIZATION TESTS ====================

func TestHeaderList_JSONSerialization(t *testing.T) {
	h := http2.NewHeaderList()
	h.Add("content-type", "application/json")
	h.AddSensitive("authorization", "Bearer secret")

	jsonBytes, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Unmarshal back
	h2 := http2.NewHeaderList()
	err = json.Unmarshal(jsonBytes, h2)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if h2.Len() != 2 {
		t.Errorf("Expected 2 headers after unmarshal, got %d", h2.Len())
	}
}

// ==================== HTTP/2 RAW PARSING TESTS ====================

func TestHTTP2Parse_HTTPStyleFormat(t *testing.T) {
	// Test HTTP/1.1-style format with HTTP/2 version
	raw := []byte("GET /api/test HTTP/2\r\nHost: example.com\r\nAccept: application/json\r\nUser-Agent: test-client\r\n\r\n")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method not parsed correctly: expected 'GET', got '%s'", req.Method)
	}

	if req.Path != "/api/test" {
		t.Errorf("Path not parsed correctly: expected '/api/test', got '%s'", req.Path)
	}

	if req.Authority != "example.com" {
		t.Errorf("Authority not set from Host header: expected 'example.com', got '%s'", req.Authority)
	}

	if req.Scheme != "https" {
		t.Errorf("Scheme should default to 'https': got '%s'", req.Scheme)
	}

	if req.Headers.Get("Accept") != "application/json" {
		t.Errorf("Accept header not parsed: got '%s'", req.Headers.Get("Accept"))
	}

	if req.Headers.Get("User-Agent") != "test-client" {
		t.Errorf("User-Agent header not parsed: got '%s'", req.Headers.Get("User-Agent"))
	}
}

func TestHTTP2Parse_HTTPStyleFormatWithBody(t *testing.T) {
	raw := []byte("POST /api/users HTTP/2\r\nHost: api.example.com\r\nContent-Type: application/json\r\n\r\n{\"name\":\"test\"}")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Method not parsed correctly: got '%s'", req.Method)
	}

	if req.Authority != "api.example.com" {
		t.Errorf("Authority not parsed correctly: got '%s'", req.Authority)
	}

	expectedBody := `{"name":"test"}`
	if string(req.Body) != expectedBody {
		t.Errorf("Body not parsed correctly: expected '%s', got '%s'", expectedBody, string(req.Body))
	}

	if req.EndStream {
		t.Error("EndStream should be false when body is present")
	}
}

func TestHTTP2Parse_PseudoHeaderFormat(t *testing.T) {
	raw := []byte(":method: GET\r\n:scheme: https\r\n:authority: example.com\r\n:path: /api/test\r\naccept: application/json\r\n\r\n")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method not parsed: expected 'GET', got '%s'", req.Method)
	}

	if req.Scheme != "https" {
		t.Errorf("Scheme not parsed: expected 'https', got '%s'", req.Scheme)
	}

	if req.Authority != "example.com" {
		t.Errorf("Authority not parsed: expected 'example.com', got '%s'", req.Authority)
	}

	if req.Path != "/api/test" {
		t.Errorf("Path not parsed: expected '/api/test', got '%s'", req.Path)
	}

	if req.Headers.Get("accept") != "application/json" {
		t.Errorf("Regular header not parsed: got '%s'", req.Headers.Get("accept"))
	}
}

func TestHTTP2Parse_PseudoHeaderFormatWithBody(t *testing.T) {
	raw := []byte(":method: POST\r\n:scheme: https\r\n:authority: api.example.com\r\n:path: /users\r\ncontent-type: application/json\r\n\r\n{\"id\":123}")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Method not parsed: got '%s'", req.Method)
	}

	expectedBody := `{"id":123}`
	if string(req.Body) != expectedBody {
		t.Errorf("Body not parsed: expected '%s', got '%s'", expectedBody, string(req.Body))
	}
}

func TestHTTP2Parse_LFLineEndings(t *testing.T) {
	// Test with LF line endings instead of CRLF
	raw := []byte("GET /test HTTP/2\nHost: example.com\nAccept: */*\n\nbody content")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed with LF endings: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method not parsed with LF: got '%s'", req.Method)
	}

	if req.Path != "/test" {
		t.Errorf("Path not parsed with LF: got '%s'", req.Path)
	}

	if string(req.Body) != "body content" {
		t.Errorf("Body not parsed with LF: got '%s'", string(req.Body))
	}
}

func TestHTTP2Parse_BuildAsHTTP1_Roundtrip(t *testing.T) {
	// Parse HTTP/2 request and build as HTTP/1.1
	raw := []byte("GET /api/v1/users HTTP/2\r\nHost: api.example.com\r\nAccept: application/json\r\nAuthorization: Bearer token123\r\n\r\n")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Build as HTTP/1.1
	http1 := req.BuildAsHTTP1()
	http1Str := string(http1)

	// Verify HTTP/1.1 format
	if !strings.HasPrefix(http1Str, "GET /api/v1/users HTTP/1.1") {
		t.Errorf("BuildAsHTTP1 should produce HTTP/1.1 request line, got: %s", http1Str[:50])
	}

	if !strings.Contains(http1Str, "Host: api.example.com") {
		t.Error("BuildAsHTTP1 should contain Host header")
	}

	if !strings.Contains(http1Str, "Accept: application/json") {
		t.Error("BuildAsHTTP1 should preserve Accept header")
	}

	if !strings.Contains(http1Str, "Authorization: Bearer token123") {
		t.Error("BuildAsHTTP1 should preserve Authorization header")
	}
}

func TestHTTP2Parse_BuildHTTP1Style_PreservesVersion(t *testing.T) {
	raw := []byte("POST /submit HTTP/2\r\nHost: example.com\r\nContent-Type: text/plain\r\n\r\ntest body")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// BuildHTTP1Style should show HTTP/2 in version
	http2Style := req.BuildHTTP1Style()
	http2StyleStr := string(http2Style)

	if !strings.HasPrefix(http2StyleStr, "POST /submit HTTP/2") {
		t.Errorf("BuildHTTP1Style should preserve HTTP/2 version, got: %s", http2StyleStr[:30])
	}
}

func TestHTTP2Parse_Build_PseudoHeaders(t *testing.T) {
	raw := []byte("GET /test HTTP/2\r\nHost: example.com\r\n\r\n")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Build() should output pseudo-header format
	built := req.Build()
	builtStr := string(built)

	if !strings.Contains(builtStr, ":method: GET") {
		t.Error("Build should contain :method pseudo-header")
	}

	if !strings.Contains(builtStr, ":path: /test") {
		t.Error("Build should contain :path pseudo-header")
	}

	if !strings.Contains(builtStr, ":authority: example.com") {
		t.Error("Build should contain :authority pseudo-header")
	}

	if !strings.Contains(builtStr, ":scheme: https") {
		t.Error("Build should contain :scheme pseudo-header")
	}
}

func TestHTTP2Parse_SkipsConnectionHeaders(t *testing.T) {
	raw := []byte("GET /test HTTP/2\r\nHost: example.com\r\nConnection: keep-alive\r\nKeep-Alive: timeout=5\r\nTransfer-Encoding: chunked\r\nUpgrade: h2c\r\nProxy-Connection: keep-alive\r\nAccept: */*\r\n\r\n")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Connection-specific headers should be skipped
	if req.Headers.Has("Connection") {
		t.Error("Connection header should be skipped")
	}

	if req.Headers.Has("Keep-Alive") {
		t.Error("Keep-Alive header should be skipped")
	}

	if req.Headers.Has("Transfer-Encoding") {
		t.Error("Transfer-Encoding header should be skipped")
	}

	if req.Headers.Has("Upgrade") {
		t.Error("Upgrade header should be skipped")
	}

	if req.Headers.Has("Proxy-Connection") {
		t.Error("Proxy-Connection header should be skipped")
	}

	// Accept should be preserved
	if !req.Headers.Has("Accept") {
		t.Error("Accept header should be preserved")
	}
}

func TestHTTP2Parse_MissingPseudoHeaders_Error(t *testing.T) {
	// Missing :method and :path in pseudo-header format
	raw := []byte(":scheme: https\r\n:authority: example.com\r\n\r\n")

	_, err := http2.Parse(raw)
	if err == nil {
		t.Error("Parse should fail when required pseudo-headers are missing")
	}
}

func TestHTTP2Parse_NotHTTP2_Error(t *testing.T) {
	// HTTP/1.1 version should fail
	raw := []byte("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n")

	_, err := http2.Parse(raw)
	if err == nil {
		t.Error("Parse should fail for HTTP/1.1 requests")
	}
}

func TestHTTP2Parse_EmptyData_Error(t *testing.T) {
	_, err := http2.Parse([]byte{})
	if err == nil {
		t.Error("Parse should fail for empty data")
	}
}

// ==================== HTTP/2 RESPONSE PARSING TESTS ====================

func TestHTTP2ParseResponse_HTTPStyleFormat(t *testing.T) {
	raw := []byte("HTTP/2 200 OK\r\nContent-Type: application/json\r\nServer: test-server\r\n\r\n{\"status\":\"ok\"}")

	resp, err := http2.ParseResponse(raw)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Status not parsed correctly: expected 200, got %d", resp.Status)
	}

	if resp.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type not parsed: got '%s'", resp.Headers.Get("Content-Type"))
	}

	if resp.Headers.Get("Server") != "test-server" {
		t.Errorf("Server header not parsed: got '%s'", resp.Headers.Get("Server"))
	}

	expectedBody := `{"status":"ok"}`
	if string(resp.Body) != expectedBody {
		t.Errorf("Body not parsed: expected '%s', got '%s'", expectedBody, string(resp.Body))
	}
}

func TestHTTP2ParseResponse_PseudoHeaderFormat(t *testing.T) {
	raw := []byte(":status: 201\r\ncontent-type: application/json\r\nlocation: /users/123\r\n\r\n{\"id\":123}")

	resp, err := http2.ParseResponse(raw)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if resp.Status != 201 {
		t.Errorf("Status not parsed: expected 201, got %d", resp.Status)
	}

	if resp.Headers.Get("location") != "/users/123" {
		t.Errorf("Location header not parsed: got '%s'", resp.Headers.Get("location"))
	}

	expectedBody := `{"id":123}`
	if string(resp.Body) != expectedBody {
		t.Errorf("Body not parsed: expected '%s', got '%s'", expectedBody, string(resp.Body))
	}
}

func TestHTTP2ParseResponse_BuildAsHTTP1(t *testing.T) {
	raw := []byte("HTTP/2 404 Not Found\r\nContent-Type: text/html\r\n\r\n<h1>Not Found</h1>")

	resp, err := http2.ParseResponse(raw)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	http1 := resp.BuildAsHTTP1()
	http1Str := string(http1)

	if !strings.HasPrefix(http1Str, "HTTP/1.1 404 Not Found") {
		t.Errorf("BuildAsHTTP1 should produce HTTP/1.1 status line, got: %s", http1Str[:30])
	}

	if !strings.Contains(http1Str, "<h1>Not Found</h1>") {
		t.Error("BuildAsHTTP1 should preserve body")
	}
}

func TestHTTP2ParseResponse_MissingStatus_Error(t *testing.T) {
	// Missing :status in pseudo-header format
	raw := []byte(":content-type: text/html\r\n\r\n")

	_, err := http2.ParseResponse(raw)
	if err == nil {
		t.Error("ParseResponse should fail when :status is missing")
	}
}

// ==================== HEADER ORDER PRESERVATION IN PARSING TESTS ====================

func TestHTTP2Parse_PreservesHeaderOrder(t *testing.T) {
	raw := []byte("GET /test HTTP/2\r\nHost: example.com\r\nFirst: 1\r\nSecond: 2\r\nThird: 3\r\nFourth: 4\r\n\r\n")

	req, err := http2.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expectedOrder := []string{"First", "Second", "Third", "Fourth"}
	headers := req.Headers.All()

	if len(headers) != len(expectedOrder) {
		t.Fatalf("Expected %d headers, got %d", len(expectedOrder), len(headers))
	}

	for i, expected := range expectedOrder {
		if headers[i].Name != expected {
			t.Errorf("Header order not preserved at position %d: expected '%s', got '%s'", i, expected, headers[i].Name)
		}
	}
}

// ==================== STREAMING PARSE TESTS ====================

func TestHTTP2ParseHeadersFromReader(t *testing.T) {
	raw := "GET /streaming HTTP/2\r\nHost: example.com\r\nContent-Length: 11\r\n\r\nHello World"
	reader := strings.NewReader(raw)

	req, bodyReader, err := http2.ParseHeadersFromReader(reader)
	if err != nil {
		t.Fatalf("ParseHeadersFromReader failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method not parsed: got '%s'", req.Method)
	}

	if req.Path != "/streaming" {
		t.Errorf("Path not parsed: got '%s'", req.Path)
	}

	// Read body from returned reader
	bodyBytes := make([]byte, 11)
	n, err := bodyReader.Read(bodyBytes)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Failed to read body: %v", err)
	}

	if n != 11 || string(bodyBytes) != "Hello World" {
		t.Errorf("Body not readable from returned reader: got '%s'", string(bodyBytes[:n]))
	}
}
