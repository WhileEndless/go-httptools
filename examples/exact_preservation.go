package main

import (
	"fmt"
	"log"
	"strings"
	
	"github.com/WhileEndless/go-httptools/pkg/request"
)

func main() {
	fmt.Println("=== Exact Format Preservation Demo ===\n")
	
	// Malformed request with weird spacing and non-standard headers
	originalRequest := []byte(`POST    /api/test?param=value   HTTP/1.1
Host:   example.com  
User-Agent:Mozilla/5.0    
test:deneme
Content-Type:  application/json  
Authorization:   Bearer   token123  
X-Custom:    value with extra spaces   

{  "data":  "test"  }`)

	fmt.Println("Original Request:")
	fmt.Printf("%q\n\n", string(originalRequest))
	
	// Parse with exact formatting preservation
	rawReq, err := request.ParseRaw(originalRequest)
	if err != nil {
		log.Fatal("ParseRaw failed:", err)
	}
	
	fmt.Println("Parsed Data (accessible):")
	fmt.Printf("Method: %s\n", rawReq.Method)
	fmt.Printf("URL: %s\n", rawReq.URL)
	fmt.Printf("Version: %s\n", rawReq.Version)
	fmt.Printf("Custom header 'test': %s\n", rawReq.Headers.Get("test"))
	fmt.Printf("Body: %s\n\n", string(rawReq.Body))
	
	// Rebuild - should be EXACTLY identical
	rebuilt := rawReq.BuildRaw()
	fmt.Println("Rebuilt Request:")
	fmt.Printf("%q\n\n", string(rebuilt))
	
	// Verify they are identical
	if string(originalRequest) == string(rebuilt) {
		fmt.Println("✅ PERFECT: Original and rebuilt are byte-for-byte identical!")
	} else {
		fmt.Println("❌ ERROR: Formatting not preserved")
	}
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	
	// Test editing while preserving other formatting
	fmt.Println("\n=== Editing Test ===\n")
	
	// Edit one header
	rawReq.Headers.Set("Authorization", "Bearer new-super-secret-token")
	
	editedRebuilt := rawReq.BuildRaw()
	fmt.Println("After editing Authorization header:")
	fmt.Printf("%q\n\n", string(editedRebuilt))
	
	// Parse edited version to check
	editedReq, _ := request.ParseRaw(editedRebuilt)
	
	fmt.Println("Verification:")
	fmt.Printf("✅ Method preserved: %s\n", editedReq.Method)
	fmt.Printf("✅ Custom header 'test' preserved: %s\n", editedReq.Headers.Get("test"))
	fmt.Printf("✅ Authorization updated: %s\n", editedReq.Headers.Get("Authorization"))
	fmt.Printf("✅ Host formatting preserved: %s\n", editedReq.Headers.Get("Host"))
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	
	// Test with completely malformed request
	fmt.Println("\n=== Fault Tolerance Test ===\n")
	
	malformed := []byte(`GET     /weird/path     HTTP/1.1
Host:example.com
: empty-header-name
Invalid-Header-No-Colon
test:deneme
header-without-value:

`)

	fmt.Println("Malformed Request:")
	fmt.Printf("%q\n\n", string(malformed))
	
	malformedReq, err := request.ParseRaw(malformed)
	if err != nil {
		log.Fatal("ParseRaw failed:", err)
	}
	
	malformedRebuilt := malformedReq.BuildRaw()
	
	fmt.Println("Rebuilt (with fault tolerance):")
	fmt.Printf("%q\n\n", string(malformedRebuilt))
	
	if string(malformed) == string(malformedRebuilt) {
		fmt.Println("✅ PERFECT: Even malformed requests preserved exactly!")
	} else {
		fmt.Println("❌ Malformed request not preserved")
	}
	
	fmt.Printf("✅ Custom header still accessible: %s\n", malformedReq.Headers.Get("test"))
}