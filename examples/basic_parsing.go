package main

import (
	"fmt"
	"log"

	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
)

func main() {
	// Example 1: Parse HTTP Request with non-standard header
	rawRequest := []byte(`GET /api/users?page=1 HTTP/1.1
Host: example.com
User-Agent: Mozilla/5.0
test:deneme
Authorization: Bearer token123
Custom-Header: value with spaces

`)

	req, err := request.Parse(rawRequest)
	if err != nil {
		log.Fatal("Failed to parse request:", err)
	}

	fmt.Println("=== Parsed Request ===")
	fmt.Printf("Method: %s\n", req.Method)
	fmt.Printf("URL: %s\n", req.URL)
	fmt.Printf("Version: %s\n", req.Version)
	fmt.Println("Headers:")
	for _, header := range req.Headers.All() {
		fmt.Printf("  %s: %s\n", header.Name, header.Value)
	}
	fmt.Printf("Body length: %d bytes\n\n", len(req.Body))

	// Example 2: Parse HTTP Response with compression
	rawResponse := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Content-Encoding: gzip
Content-Length: 123
Server: nginx/1.18.0

` + string([]byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}) + `{"message":"Hello World"}`)

	resp, err := response.Parse(rawResponse)
	if err != nil {
		log.Fatal("Failed to parse response:", err)
	}

	fmt.Println("=== Parsed Response ===")
	fmt.Printf("Version: %s\n", resp.Version)
	fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.StatusText)
	fmt.Printf("Compressed: %t\n", resp.Compressed)
	fmt.Println("Headers:")
	for _, header := range resp.Headers.All() {
		fmt.Printf("  %s: %s\n", header.Name, header.Value)
	}
	fmt.Printf("Body (decompressed): %s\n", string(resp.Body))
	fmt.Printf("Raw body length: %d bytes\n", len(resp.RawBody))
}
