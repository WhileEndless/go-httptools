package main

import (
	"fmt"

	"github.com/WhileEndless/go-httptools/pkg/response"
)

func main() {
	// Example HTTP response with chunked transfer encoding
	rawResponse := []byte("HTTP/1.1 200 OK\r\n" +
		"Content-Type: application/json\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n" +
		"1c\r\n" +
		"{\"message\":\"Hello, \"}\r\n" +
		"e\r\n" +
		"{\"name\":\"World\"}\r\n" +
		"0\r\n" +
		"\r\n")

	fmt.Println("=== Example 1: Default Behavior (Chunked Body Preserved) ===")
	resp1, err := response.Parse(rawResponse)
	if err != nil {
		panic(err)
	}

	fmt.Printf("IsBodyChunked: %t\n", resp1.IsBodyChunked)
	fmt.Printf("Transfer-Encoding: %s\n", resp1.Headers.Get("Transfer-Encoding"))
	fmt.Printf("Body (raw chunked): %q\n", string(resp1.Body))
	fmt.Println()

	fmt.Println("=== Example 2: Auto-Decode Chunked Encoding ===")
	opts := response.ParseOptions{
		AutoDecodeChunked: true,
	}

	resp2, err := response.ParseWithOptions(rawResponse, opts)
	if err != nil {
		panic(err)
	}

	fmt.Printf("IsBodyChunked: %t\n", resp2.IsBodyChunked)
	fmt.Printf("Transfer-Encoding: %s\n", resp2.Headers.Get("Transfer-Encoding"))
	fmt.Printf("Content-Length: %s\n", resp2.Headers.Get("Content-Length"))
	fmt.Printf("Body (decoded): %q\n", string(resp2.Body))
	fmt.Printf("RawBody (original chunked): %q\n", string(resp2.RawBody))
	fmt.Println()

	// Example with trailers
	rawWithTrailers := []byte("HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n" +
		"7\r\n" +
		"Mozilla\r\n" +
		"9\r\n" +
		"Developer\r\n" +
		"7\r\n" +
		"Network\r\n" +
		"0\r\n" +
		"X-Checksum: abc123\r\n" +
		"X-Custom-Trailer: value\r\n" +
		"\r\n")

	fmt.Println("=== Example 3: Auto-Decode with Trailers Preservation ===")
	optsWithTrailers := response.ParseOptions{
		AutoDecodeChunked:       true,
		PreserveChunkedTrailers: true,
	}

	resp3, err := response.ParseWithOptions(rawWithTrailers, optsWithTrailers)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Body (decoded): %q\n", string(resp3.Body))
	fmt.Printf("Trailers preserved as headers:\n")
	fmt.Printf("  X-Checksum: %s\n", resp3.Headers.Get("X-Checksum"))
	fmt.Printf("  X-Custom-Trailer: %s\n", resp3.Headers.Get("X-Custom-Trailer"))
	fmt.Println()

	fmt.Println("=== Example 4: Manual Decoding (Alternative Approach) ===")
	resp4, err := response.Parse(rawResponse)
	if err != nil {
		panic(err)
	}

	if resp4.IsBodyChunked {
		fmt.Println("Body is chunked, decoding manually...")
		trailers := resp4.DecodeChunkedBody()
		fmt.Printf("Decoded body: %q\n", string(resp4.Body))
		fmt.Printf("Trailers found: %v\n", trailers)
	}
}
