package main

import (
	"fmt"
	"log"
	"strings"
	
	"github.com/WhileEndless/go-httptools/pkg/request"
)

func main() {
	fmt.Println("=== Header Positioning Demo ===\n")
	
	// Original request
	original := []byte(`POST /api/test HTTP/1.1
Host: example.com
User-Agent: Mozilla/5.0
test:deneme
Content-Type: application/json

{"data": "test"}`)

	fmt.Println("Original Request:")
	rawReq, err := request.ParseRaw(original)
	if err != nil {
		log.Fatal(err)
	}
	
	for i, header := range rawReq.Headers.All() {
		fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
	}
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	
	// Test 1: SetAfter - Auth header Host headerından sonra
	fmt.Println("\n1. SetAfter - Auth header Host'tan sonra:")
	rawReq.Headers.SetAfter("Authorization", "Bearer token123", "Host")
	
	for i, header := range rawReq.Headers.All() {
		fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
	}
	
	rebuilt1 := rawReq.BuildRawString()
	fmt.Printf("\nRebuilt:\n%s\n", rebuilt1)
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	
	// Test 2: SetBefore - X-Custom header test headerından önce
	fmt.Println("\n2. SetBefore - X-Custom header test'ten önce:")
	rawReq.Headers.SetBefore("X-Custom", "custom-value", "test")
	
	for i, header := range rawReq.Headers.All() {
		fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
	}
	
	rebuilt2 := rawReq.BuildRawString()
	fmt.Printf("\nRebuilt:\n%s\n", rebuilt2)
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	
	// Test 3: SetAt - Cookie header index 1'e (Host'tan hemen sonra)
	fmt.Println("\n3. SetAt - Cookie header index 1'e (Host'tan hemen sonra):")
	rawReq.Headers.SetAt("Cookie", "session=abc123", 1)
	
	for i, header := range rawReq.Headers.All() {
		fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
	}
	
	rebuilt3 := rawReq.BuildRawString()
	fmt.Printf("\nRebuilt:\n%s\n", rebuilt3)
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	
	// Test 4: Normal Set - sonda eklenir
	fmt.Println("\n4. Normal Set - API-Key header sonda:")
	rawReq.Headers.Set("API-Key", "secret123")
	
	for i, header := range rawReq.Headers.All() {
		fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
	}
	
	rebuilt4 := rawReq.BuildRawString()
	fmt.Printf("\nRebuilt:\n%s\n", rebuilt4)
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	
	// Test 5: Update existing header - pozisyon değişmez
	fmt.Println("\n5. Update Existing - Authorization header güncelleme:")
	rawReq.Headers.SetAfter("Authorization", "Bearer updated-token", "Host")
	
	for i, header := range rawReq.Headers.All() {
		fmt.Printf("%d. %s: %s\n", i+1, header.Name, header.Value)
	}
	
	rebuilt5 := rawReq.BuildRawString()
	fmt.Printf("\nRebuilt:\n%s\n", rebuilt5)
	
	fmt.Println("\n✅ Custom header 'test' hala erişilebilir:", rawReq.Headers.Get("test"))
}