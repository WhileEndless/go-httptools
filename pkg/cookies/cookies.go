package cookies

import (
	"fmt"
	"strings"
)

// Cookie represents a request cookie (from Cookie header)
type Cookie struct {
	Name  string
	Value string
}

// ParseCookies parses Cookie header value
// Never fails - returns empty slice if malformed
// Format: "name1=value1; name2=value2; name3=value3"
func ParseCookies(cookieHeader string) []Cookie {
	if cookieHeader == "" {
		return []Cookie{}
	}

	var cookies []Cookie

	// Split by semicolon
	parts := strings.Split(cookieHeader, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by equals sign
		idx := strings.Index(part, "=")
		if idx == -1 {
			// No equals sign, treat whole thing as name with empty value
			cookies = append(cookies, Cookie{
				Name:  part,
				Value: "",
			})
			continue
		}

		name := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])

		// Remove quotes if present
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		cookies = append(cookies, Cookie{
			Name:  name,
			Value: value,
		})
	}

	return cookies
}

// BuildCookieHeader builds Cookie header from cookies
// Format: "name1=value1; name2=value2"
func BuildCookieHeader(cookies []Cookie) string {
	if len(cookies) == 0 {
		return ""
	}

	var parts []string
	for _, cookie := range cookies {
		if cookie.Name == "" {
			continue
		}
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}

	return strings.Join(parts, "; ")
}

// ResponseCookie represents Set-Cookie header (from HTTP response)
type ResponseCookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	Expires  string
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite string
	Raw      string // Original Set-Cookie header (preserved)
}

// ParseSetCookie parses Set-Cookie header
// Never fails - best effort parse
// Format: "name=value; Path=/; Domain=.example.com; Expires=...; Max-Age=3600; Secure; HttpOnly; SameSite=Strict"
func ParseSetCookie(setCookie string) ResponseCookie {
	cookie := ResponseCookie{
		Raw:    setCookie,
		MaxAge: -1, // -1 means not set
	}

	if setCookie == "" {
		return cookie
	}

	// Split by semicolon
	parts := strings.Split(setCookie, ";")

	// First part is name=value
	if len(parts) > 0 {
		firstPart := strings.TrimSpace(parts[0])
		idx := strings.Index(firstPart, "=")
		if idx != -1 {
			cookie.Name = strings.TrimSpace(firstPart[:idx])
			cookie.Value = strings.TrimSpace(firstPart[idx+1:])

			// Remove quotes if present
			if len(cookie.Value) >= 2 && cookie.Value[0] == '"' && cookie.Value[len(cookie.Value)-1] == '"' {
				cookie.Value = cookie.Value[1 : len(cookie.Value)-1]
			}
		} else {
			// No equals sign in first part, use whole thing as name
			cookie.Name = firstPart
		}
	}

	// Parse attributes
	for i := 1; i < len(parts); i++ {
		attr := strings.TrimSpace(parts[i])
		if attr == "" {
			continue
		}

		// Check for key=value attributes
		idx := strings.Index(attr, "=")
		if idx != -1 {
			key := strings.ToLower(strings.TrimSpace(attr[:idx]))
			value := strings.TrimSpace(attr[idx+1:])

			switch key {
			case "path":
				cookie.Path = value
			case "domain":
				cookie.Domain = value
			case "expires":
				cookie.Expires = value
			case "max-age":
				// Try to parse as int
				var maxAge int
				if _, err := fmt.Sscanf(value, "%d", &maxAge); err == nil {
					cookie.MaxAge = maxAge
				}
			case "samesite":
				cookie.SameSite = value
			}
		} else {
			// Boolean attributes
			attrLower := strings.ToLower(attr)
			switch attrLower {
			case "secure":
				cookie.Secure = true
			case "httponly":
				cookie.HttpOnly = true
			}
		}
	}

	return cookie
}

// Build rebuilds Set-Cookie header from ResponseCookie
// Returns the original Raw header if it exists and no modifications were made
func (c *ResponseCookie) Build() string {
	var parts []string

	// Name=Value
	if c.Name != "" {
		parts = append(parts, c.Name+"="+c.Value)
	}

	// Path
	if c.Path != "" {
		parts = append(parts, "Path="+c.Path)
	}

	// Domain
	if c.Domain != "" {
		parts = append(parts, "Domain="+c.Domain)
	}

	// Expires
	if c.Expires != "" {
		parts = append(parts, "Expires="+c.Expires)
	}

	// Max-Age (only if explicitly set to positive value)
	if c.MaxAge > 0 {
		parts = append(parts, fmt.Sprintf("Max-Age=%d", c.MaxAge))
	}

	// Secure
	if c.Secure {
		parts = append(parts, "Secure")
	}

	// HttpOnly
	if c.HttpOnly {
		parts = append(parts, "HttpOnly")
	}

	// SameSite
	if c.SameSite != "" {
		parts = append(parts, "SameSite="+c.SameSite)
	}

	return strings.Join(parts, "; ")
}
