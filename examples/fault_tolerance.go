package main

import (
	"fmt"
	"log"
	
	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
	"github.com/WhileEndless/go-httptools/pkg/utils"
)

func main() {
	fmt.Println("=== Fault Tolerance Examples ===\n")
	
	// Example 1: Malformed request with invalid headers
	malformedRequest := []byte(`GET /path HTTP/1.1
Host: example.com
: empty-header-name
Invalid-Header-No-Colon
test:deneme
X-Custom: normal-header

body content here`)

	req, err := request.Parse(malformedRequest)
	if err != nil {
		fmt.Printf("Parse error (expected): %v\n", err)
	} else {
		fmt.Println("=== Parsed Malformed Request ===")
		fmt.Printf("Method: %s, URL: %s\n", req.Method, req.URL)
		fmt.Println("Headers (with fault tolerance):")
		for _, header := range req.Headers.All() {
			fmt.Printf("  '%s': '%s'\n", header.Name, header.Value)
		}
		
		// Validate to see issues
		validation := utils.ValidateRequest(req)
		fmt.Printf("Valid: %t\n", validation.Valid)
		if len(validation.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, warning := range validation.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}
		fmt.Println()
	}
	
	// Example 2: Response with invalid status and missing version
	malformedResponse := []byte(`HTTP/1.1 999 Custom Status
Content-Type: text/plain
Content-Length: 5
test:deneme

Hello`)

	resp, err := response.Parse(malformedResponse)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
	} else {
		fmt.Println("=== Parsed Response with Invalid Status ===")
		fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.StatusText)
		fmt.Printf("Version: %s\n", resp.Version)
		
		// Validate response
		validation := utils.ValidateResponse(resp)
		fmt.Printf("Valid: %t\n", validation.Valid)
		if len(validation.Errors) > 0 {
			fmt.Println("Errors:")
			for _, error := range validation.Errors {
				fmt.Printf("  - %s\n", error)
			}
		}
		if len(validation.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, warning := range validation.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}
		fmt.Println()
	}
	
	// Example 3: Request with minimal format
	minimalRequest := []byte(`GET /`)
	
	req2, err := request.Parse(minimalRequest)
	if err != nil {
		fmt.Printf("Minimal request parse error: %v\n", err)
	} else {
		fmt.Println("=== Parsed Minimal Request ===")
		fmt.Printf("Method: %s, URL: %s, Version: %s\n", req2.Method, req2.URL, req2.Version)
		fmt.Println("Headers count:", req2.Headers.Len())
		
		// Build back to see fault tolerance
		rebuilt := req2.BuildString()
		fmt.Println("Rebuilt:")
		fmt.Printf("%q\n", rebuilt)
	}
	
	// Example 4: Preserve order even with duplicate headers
	duplicateHeaders := []byte(`POST /test HTTP/1.1
Host: example.com
X-Custom: first-value
Content-Type: application/json
X-Custom: second-value
test:deneme

{"data": "test"}`)

	req3, err := request.Parse(duplicateHeaders)
	if err != nil {
		log.Fatal("Failed to parse:", err)
	}
	
	fmt.Println("\n=== Header Order Preservation ===")
	fmt.Println("All headers in original order:")
	for i, header := range req3.Headers.All() {
		fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
	}
	
	// Show that lookup gets the last value
	fmt.Printf("X-Custom header value (last wins): %s\n", req3.Headers.Get("X-Custom"))
	fmt.Printf("test header value: %s\n", req3.Headers.Get("test"))
}