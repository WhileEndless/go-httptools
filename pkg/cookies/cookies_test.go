package cookies

import (
	"testing"
)

// ============================================================================
// Request Cookie Tests
// ============================================================================

func TestParseCookies_Simple(t *testing.T) {
	input := "session=abc123; user=john"
	cookies := ParseCookies(input)

	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].Name != "session" || cookies[0].Value != "abc123" {
		t.Errorf("Expected session=abc123, got %s=%s", cookies[0].Name, cookies[0].Value)
	}

	if cookies[1].Name != "user" || cookies[1].Value != "john" {
		t.Errorf("Expected user=john, got %s=%s", cookies[1].Name, cookies[1].Value)
	}
}

func TestParseCookies_SingleCookie(t *testing.T) {
	input := "token=xyz"
	cookies := ParseCookies(input)

	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	if cookies[0].Name != "token" || cookies[0].Value != "xyz" {
		t.Errorf("Expected token=xyz, got %s=%s", cookies[0].Name, cookies[0].Value)
	}
}

func TestParseCookies_WithSpaces(t *testing.T) {
	input := "  name1  =  value1  ;  name2  =  value2  "
	cookies := ParseCookies(input)

	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].Name != "name1" || cookies[0].Value != "value1" {
		t.Errorf("Expected name1=value1, got %s=%s", cookies[0].Name, cookies[0].Value)
	}
}

func TestParseCookies_WithQuotes(t *testing.T) {
	input := `session="abc123"; user="john doe"`
	cookies := ParseCookies(input)

	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies, got %d", len(cookies))
	}

	// Quotes should be removed
	if cookies[0].Value != "abc123" {
		t.Errorf("Expected value abc123, got %s", cookies[0].Value)
	}

	if cookies[1].Value != "john doe" {
		t.Errorf("Expected value 'john doe', got %s", cookies[1].Value)
	}
}

func TestParseCookies_Empty(t *testing.T) {
	input := ""
	cookies := ParseCookies(input)

	if len(cookies) != 0 {
		t.Errorf("Expected empty slice, got %d cookies", len(cookies))
	}
}

func TestParseCookies_Malformed_NoEquals(t *testing.T) {
	input := "invalidcookie"
	cookies := ParseCookies(input) // Should not panic

	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	// Should treat whole thing as name with empty value
	if cookies[0].Name != "invalidcookie" {
		t.Errorf("Expected name=invalidcookie, got %s", cookies[0].Name)
	}

	if cookies[0].Value != "" {
		t.Errorf("Expected empty value, got %s", cookies[0].Value)
	}
}

func TestParseCookies_Malformed_TrailingSemicolon(t *testing.T) {
	input := "name1=value1; name2=value2;"
	cookies := ParseCookies(input) // Should not panic

	if len(cookies) != 2 {
		t.Errorf("Expected 2 cookies, got %d", len(cookies))
	}
}

func TestBuildCookieHeader_Simple(t *testing.T) {
	cookies := []Cookie{
		{Name: "session", Value: "abc123"},
		{Name: "user", Value: "john"},
	}

	result := BuildCookieHeader(cookies)
	expected := "session=abc123; user=john"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestBuildCookieHeader_Empty(t *testing.T) {
	cookies := []Cookie{}
	result := BuildCookieHeader(cookies)

	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestBuildCookieHeader_SkipEmptyName(t *testing.T) {
	cookies := []Cookie{
		{Name: "valid", Value: "abc"},
		{Name: "", Value: "should_skip"},
		{Name: "another", Value: "xyz"},
	}

	result := BuildCookieHeader(cookies)
	expected := "valid=abc; another=xyz"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestCookieRoundTrip(t *testing.T) {
	testCases := []string{
		"session=abc123",
		"a=1; b=2; c=3",
		"token=xyz; user=john; logged_in=true",
	}

	for _, original := range testCases {
		cookies := ParseCookies(original)
		rebuilt := BuildCookieHeader(cookies)

		if rebuilt != original {
			t.Errorf("Round-trip failed: original=%q, rebuilt=%q", original, rebuilt)
		}
	}
}

// ============================================================================
// Response Set-Cookie Tests
// ============================================================================

func TestParseSetCookie_Simple(t *testing.T) {
	input := "session=abc123"
	cookie := ParseSetCookie(input)

	if cookie.Name != "session" {
		t.Errorf("Expected name=session, got %s", cookie.Name)
	}

	if cookie.Value != "abc123" {
		t.Errorf("Expected value=abc123, got %s", cookie.Value)
	}

	if cookie.Raw != input {
		t.Errorf("Expected Raw to be preserved")
	}
}

func TestParseSetCookie_WithAttributes(t *testing.T) {
	input := "id=a3fWa; Expires=Wed, 21 Oct 2025 07:28:00 GMT; Path=/; Domain=.example.com; Secure; HttpOnly"
	cookie := ParseSetCookie(input)

	if cookie.Name != "id" {
		t.Errorf("Expected name=id, got %s", cookie.Name)
	}

	if cookie.Value != "a3fWa" {
		t.Errorf("Expected value=a3fWa, got %s", cookie.Value)
	}

	if cookie.Path != "/" {
		t.Errorf("Expected Path=/, got %s", cookie.Path)
	}

	if cookie.Domain != ".example.com" {
		t.Errorf("Expected Domain=.example.com, got %s", cookie.Domain)
	}

	if !cookie.Secure {
		t.Error("Expected Secure=true")
	}

	if !cookie.HttpOnly {
		t.Error("Expected HttpOnly=true")
	}

	if cookie.Expires == "" {
		t.Error("Expected Expires to be set")
	}
}

func TestParseSetCookie_MaxAge(t *testing.T) {
	input := "token=xyz; Max-Age=3600"
	cookie := ParseSetCookie(input)

	if cookie.MaxAge != 3600 {
		t.Errorf("Expected MaxAge=3600, got %d", cookie.MaxAge)
	}
}

func TestParseSetCookie_SameSite(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"session=abc; SameSite=Strict", "Strict"},
		{"session=abc; SameSite=Lax", "Lax"},
		{"session=abc; SameSite=None", "None"},
	}

	for _, tc := range testCases {
		cookie := ParseSetCookie(tc.input)
		if cookie.SameSite != tc.expected {
			t.Errorf("For %q, expected SameSite=%s, got %s", tc.input, tc.expected, cookie.SameSite)
		}
	}
}

func TestParseSetCookie_Empty(t *testing.T) {
	input := ""
	cookie := ParseSetCookie(input)

	if cookie.Name != "" {
		t.Errorf("Expected empty name, got %s", cookie.Name)
	}
}

func TestParseSetCookie_Malformed(t *testing.T) {
	malformed := []string{
		"nocookie",
		";;;",
		"=noname",
	}

	for _, input := range malformed {
		cookie := ParseSetCookie(input) // Should not panic
		_ = cookie
	}
}

func TestResponseCookie_Build(t *testing.T) {
	cookie := ResponseCookie{
		Name:     "session",
		Value:    "abc123",
		Path:     "/",
		Domain:   ".example.com",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: "Strict",
	}

	result := cookie.Build()

	// Check that all attributes are present
	if !contains(result, "session=abc123") {
		t.Error("Missing name=value")
	}
	if !contains(result, "Path=/") {
		t.Error("Missing Path")
	}
	if !contains(result, "Domain=.example.com") {
		t.Error("Missing Domain")
	}
	if !contains(result, "Max-Age=3600") {
		t.Error("Missing Max-Age")
	}
	if !contains(result, "Secure") {
		t.Error("Missing Secure")
	}
	if !contains(result, "HttpOnly") {
		t.Error("Missing HttpOnly")
	}
	if !contains(result, "SameSite=Strict") {
		t.Error("Missing SameSite")
	}
}

func TestResponseCookie_Build_Minimal(t *testing.T) {
	cookie := ResponseCookie{
		Name:  "token",
		Value: "xyz",
	}

	result := cookie.Build()
	expected := "token=xyz"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestSetCookieRoundTrip(t *testing.T) {
	testCases := []string{
		"session=abc123",
		"id=a3fWa; Path=/; Secure; HttpOnly",
		"token=xyz; Max-Age=3600; SameSite=Lax",
	}

	for _, original := range testCases {
		cookie := ParseSetCookie(original)
		rebuilt := cookie.Build()

		// Parse both and compare (order might differ)
		originalCookie := ParseSetCookie(original)
		rebuiltCookie := ParseSetCookie(rebuilt)

		if originalCookie.Name != rebuiltCookie.Name {
			t.Errorf("Name mismatch: %s vs %s", originalCookie.Name, rebuiltCookie.Name)
		}
		if originalCookie.Value != rebuiltCookie.Value {
			t.Errorf("Value mismatch: %s vs %s", originalCookie.Value, rebuiltCookie.Value)
		}
		if originalCookie.Path != rebuiltCookie.Path {
			t.Errorf("Path mismatch: %s vs %s", originalCookie.Path, rebuiltCookie.Path)
		}
		if originalCookie.Secure != rebuiltCookie.Secure {
			t.Errorf("Secure mismatch: %v vs %v", originalCookie.Secure, rebuiltCookie.Secure)
		}
		if originalCookie.HttpOnly != rebuiltCookie.HttpOnly {
			t.Errorf("HttpOnly mismatch: %v vs %v", originalCookie.HttpOnly, rebuiltCookie.HttpOnly)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && s[1:len(substr)+1] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkParseCookies(b *testing.B) {
	input := "session=abc123; user=john; token=xyz; logged_in=true"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseCookies(input)
	}
}

func BenchmarkBuildCookieHeader(b *testing.B) {
	cookies := []Cookie{
		{Name: "session", Value: "abc123"},
		{Name: "user", Value: "john"},
		{Name: "token", Value: "xyz"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildCookieHeader(cookies)
	}
}

func BenchmarkParseSetCookie(b *testing.B) {
	input := "id=a3fWa; Expires=Wed, 21 Oct 2025 07:28:00 GMT; Path=/; Domain=.example.com; Secure; HttpOnly"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseSetCookie(input)
	}
}

func BenchmarkResponseCookieBuild(b *testing.B) {
	cookie := ResponseCookie{
		Name:     "session",
		Value:    "abc123",
		Path:     "/",
		Domain:   ".example.com",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cookie.Build()
	}
}
