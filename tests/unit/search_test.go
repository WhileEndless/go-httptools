package unit

import (
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
	"github.com/WhileEndless/go-httptools/pkg/search"
)

// ==================== SEARCH PACKAGE TESTS ====================

func TestSearcher_BasicSearch(t *testing.T) {
	opts := search.DefaultOptions()
	opts.Pattern = "hello"

	searcher, err := search.NewSearcher(opts)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}

	data := []byte("hello world, hello universe")
	results := searcher.SearchBytes(data)

	if len(results) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(results))
	}
}

func TestSearcher_CaseInsensitive(t *testing.T) {
	opts := search.DefaultOptions()
	opts.Pattern = "hello"
	opts.CaseInsensitive = true

	searcher, err := search.NewSearcher(opts)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}

	data := []byte("Hello HELLO hello HeLLo")
	results := searcher.SearchBytes(data)

	if len(results) != 4 {
		t.Errorf("Expected 4 matches, got %d", len(results))
	}
}

func TestSearcher_RegexSearch(t *testing.T) {
	opts := search.DefaultOptions()
	opts.Pattern = `\d{3}-\d{4}`
	opts.UseRegex = true

	searcher, err := search.NewSearcher(opts)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}

	data := []byte("Call 123-4567 or 890-1234 today")
	results := searcher.SearchBytes(data)

	if len(results) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(results))
	}

	if results[0].MatchedText != "123-4567" {
		t.Errorf("Expected '123-4567', got '%s'", results[0].MatchedText)
	}
}

func TestSearcher_RegexCaseInsensitive(t *testing.T) {
	opts := search.DefaultOptions()
	opts.Pattern = `error`
	opts.UseRegex = true
	opts.CaseInsensitive = true

	searcher, err := search.NewSearcher(opts)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}

	data := []byte("Error ERROR error ErRoR")
	results := searcher.SearchBytes(data)

	if len(results) != 4 {
		t.Errorf("Expected 4 matches, got %d", len(results))
	}
}

func TestSearcher_MaxResults(t *testing.T) {
	opts := search.DefaultOptions()
	opts.Pattern = "a"
	opts.MaxResults = 3

	searcher, err := search.NewSearcher(opts)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}

	data := []byte("a a a a a a a a a a")
	results := searcher.SearchBytes(data)

	if len(results) != 3 {
		t.Errorf("Expected 3 matches (max), got %d", len(results))
	}
}

func TestSearcher_LineNumber(t *testing.T) {
	opts := search.DefaultOptions()
	opts.Pattern = "error"

	searcher, err := search.NewSearcher(opts)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}

	data := []byte("line1\nline2\nerror here\nline4")
	results := searcher.SearchBytes(data)

	if len(results) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(results))
	}

	if results[0].LineNumber != 3 {
		t.Errorf("Expected line 3, got %d", results[0].LineNumber)
	}
}

func TestQuickSearch(t *testing.T) {
	data := []byte("Hello World")

	if !search.QuickSearch(data, "World", false) {
		t.Error("QuickSearch should find 'World'")
	}

	if search.QuickSearch(data, "world", false) {
		t.Error("QuickSearch should not find 'world' (case sensitive)")
	}

	if !search.QuickSearch(data, "world", true) {
		t.Error("QuickSearch should find 'world' (case insensitive)")
	}
}

func TestQuickSearchRegex(t *testing.T) {
	data := []byte("Error code: 404")

	match, err := search.QuickSearchRegex(data, `\d{3}`)
	if err != nil {
		t.Fatalf("QuickSearchRegex failed: %v", err)
	}
	if !match {
		t.Error("QuickSearchRegex should match '\\d{3}'")
	}

	match, err = search.QuickSearchRegex(data, `\d{5}`)
	if err != nil {
		t.Fatalf("QuickSearchRegex failed: %v", err)
	}
	if match {
		t.Error("QuickSearchRegex should not match '\\d{5}'")
	}
}

func TestReplaceAll_Simple(t *testing.T) {
	data := []byte("hello world, hello universe")
	opts := search.DefaultOptions()

	result, count, err := search.ReplaceAll(data, "hello", "hi", opts)
	if err != nil {
		t.Fatalf("ReplaceAll failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 replacements, got %d", count)
	}

	expected := "hi world, hi universe"
	if string(result) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestReplaceAll_Regex(t *testing.T) {
	data := []byte("Call 123-4567 or 890-1234")
	opts := search.DefaultOptions()
	opts.UseRegex = true

	result, count, err := search.ReplaceAll(data, `\d{3}-\d{4}`, "XXX-XXXX", opts)
	if err != nil {
		t.Fatalf("ReplaceAll failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 replacements, got %d", count)
	}

	expected := "Call XXX-XXXX or XXX-XXXX"
	if string(result) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestReplaceAll_CaseInsensitive(t *testing.T) {
	data := []byte("Hello HELLO hello")
	opts := search.DefaultOptions()
	opts.CaseInsensitive = true

	result, count, err := search.ReplaceAll(data, "hello", "hi", opts)
	if err != nil {
		t.Fatalf("ReplaceAll failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 replacements, got %d", count)
	}

	expected := "hi hi hi"
	if string(result) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// ==================== REQUEST SEARCH TESTS ====================

func TestRequest_Search_Basic(t *testing.T) {
	raw := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\nUser-Agent: TestAgent\r\n\r\n{\"user\":\"admin\"}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	results, err := req.Search("admin", search.DefaultOptions())
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if !results.HasMatches() {
		t.Error("Should find 'admin' in body")
	}

	if results.BodyMatches != 1 {
		t.Errorf("Expected 1 body match, got %d", results.BodyMatches)
	}
}

func TestRequest_Search_InHeaders(t *testing.T) {
	raw := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\nX-Custom: secret-value\r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	results, err := req.SearchHeaders("secret", false)
	if err != nil {
		t.Fatalf("SearchHeaders failed: %v", err)
	}

	if results.HeaderMatches != 1 {
		t.Errorf("Expected 1 header match, got %d", results.HeaderMatches)
	}
}

func TestRequest_Contains(t *testing.T) {
	raw := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n{\"password\":\"secret123\"}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !req.Contains("password", false) {
		t.Error("Should contain 'password'")
	}

	if !req.Contains("SECRET", true) {
		t.Error("Should contain 'SECRET' (case insensitive)")
	}

	if req.Contains("SECRET", false) {
		t.Error("Should not contain 'SECRET' (case sensitive)")
	}
}

func TestRequest_ContainsRegex(t *testing.T) {
	raw := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n{\"code\":\"ABC-123\"}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	match, err := req.ContainsRegex(`[A-Z]{3}-\d{3}`)
	if err != nil {
		t.Fatalf("ContainsRegex failed: %v", err)
	}

	if !match {
		t.Error("Should match pattern")
	}
}

func TestRequest_ReplaceInBody(t *testing.T) {
	raw := []byte("POST /api HTTP/1.1\r\nHost: example.com\r\n\r\n{\"old\":\"value\",\"old\":\"data\"}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	opts := search.DefaultOptions()
	count, err := req.ReplaceInBody("old", "new", opts)
	if err != nil {
		t.Fatalf("ReplaceInBody failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 replacements, got %d", count)
	}

	expected := `{"new":"value","new":"data"}`
	if string(req.Body) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, req.Body)
	}
}

// ==================== RESPONSE SEARCH TESTS ====================

func TestResponse_Search_Basic(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"status\":\"success\"}")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	results, err := resp.Search("success", search.DefaultOptions())
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if !results.HasMatches() {
		t.Error("Should find 'success' in body")
	}
}

func TestResponse_Search_InHeaders(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nX-Server: nginx/1.18\r\n\r\n{}")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	results, err := resp.SearchHeaders("nginx", false)
	if err != nil {
		t.Fatalf("SearchHeaders failed: %v", err)
	}

	if results.HeaderMatches != 1 {
		t.Errorf("Expected 1 header match, got %d", results.HeaderMatches)
	}
}

func TestResponse_Contains(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<html><body>Hello World</body></html>")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !resp.Contains("Hello", false) {
		t.Error("Should contain 'Hello'")
	}

	if !resp.Contains("html", true) {
		t.Error("Should contain 'html' (case insensitive)")
	}
}

func TestResponse_SearchRegex(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<div id=\"user-123\">User</div><div id=\"user-456\">Admin</div>")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	results, err := resp.SearchRegex(`user-\d+`)
	if err != nil {
		t.Fatalf("SearchRegex failed: %v", err)
	}

	if len(results.Results) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(results.Results))
	}
}

func TestResponse_ReplaceInBody(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<p>secret data</p><p>secret info</p>")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	opts := search.DefaultOptions()
	count, err := resp.ReplaceInBody("secret", "hidden", opts)
	if err != nil {
		t.Fatalf("ReplaceInBody failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 replacements, got %d", count)
	}

	expected := `<p>hidden data</p><p>hidden info</p>`
	if string(resp.Body) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resp.Body)
	}
}

// ==================== SEARCH HEADER TESTS ====================

func TestSearcher_SearchHeaders(t *testing.T) {
	opts := search.DefaultOptions()
	opts.Pattern = "json"
	opts.SearchHeaderNames = true
	opts.CaseInsensitive = true // Enable case insensitive to match both

	searcher, err := search.NewSearcher(opts)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}

	headers := []search.HeaderField{
		{Name: "Content-Type", Value: "application/json"},
		{Name: "Accept", Value: "text/html"},
		{Name: "X-Json-Header", Value: "value"},
	}

	results := searcher.SearchHeaders(headers)

	if len(results) != 2 {
		t.Errorf("Expected 2 matches (Content-Type value and X-Json-Header name), got %d", len(results))
	}
}

func TestSearcher_SearchHeaders_RawFormat(t *testing.T) {
	opts := search.DefaultOptions()
	opts.Pattern = "  example"
	opts.SearchHeaderRaw = true

	searcher, err := search.NewSearcher(opts)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}

	headers := []search.HeaderField{
		{Name: "Host", Value: "  example.com", OriginalLine: "Host:  example.com"},
	}

	results := searcher.SearchHeaders(headers)

	if len(results) != 1 {
		t.Errorf("Expected 1 match in raw format, got %d", len(results))
	}
}
