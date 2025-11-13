package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/WhileEndless/go-httptools/pkg/rawhttp"
)

func main() {
	fmt.Println("=== GO-RAWHTTP ÖZELLİKLERİ TEST PROGRAMI ===\n")

	// Test 1: Basic HTTP/1.1 Request
	fmt.Println("📝 Test 1: Basic HTTP/1.1 Request (example.com)")
	testBasicHTTP()

	// Test 2: HTTPS Request with TLS
	fmt.Println("\n📝 Test 2: HTTPS Request with TLS")
	testHTTPS()

	// Test 3: Connection Reuse (Keep-Alive)
	fmt.Println("\n📝 Test 3: Connection Reuse (Keep-Alive)")
	testConnectionReuse()

	// Test 4: Custom Timeouts
	fmt.Println("\n📝 Test 4: Custom Timeouts")
	testTimeouts()

	// Test 5: Raw Response Preservation
	fmt.Println("\n📝 Test 5: Raw Response Preservation")
	testRawResponsePreservation()

	// Test 6: Connection Metadata
	fmt.Println("\n📝 Test 6: Connection Metadata")
	testConnectionMetadata()

	// Test 7: InsecureSkipVerify
	fmt.Println("\n📝 Test 7: InsecureSkipVerify (TLS Configuration)")
	testInsecureSkipVerify()

	// Test 8: Error Handling
	fmt.Println("\n📝 Test 8: Error Handling (Invalid Host)")
	testErrorHandling()

	fmt.Println("\n\n✅ TÜM TESTLER TAMAMLANDI!")
}

func testBasicHTTP() {
	sender := rawhttp.NewSender()
	defer sender.Close()

	rawRequest := []byte("GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n")

	opts := rawhttp.Options{
		Scheme: "http",
		Host:   "example.com",
		Port:   80,
	}

	ctx := context.Background()
	resp, err := sender.Do(ctx, rawRequest, opts)
	if err != nil {
		log.Printf("   ❌ HATA: %v\n", err)
		return
	}

	fmt.Printf("   ✅ Status Code: %d\n", resp.StatusCode)
	fmt.Printf("   ✅ Raw Response Length: %d bytes\n", len(resp.Raw))
	fmt.Printf("   ✅ Connected IP: %s\n", resp.ConnectedIP)
	fmt.Printf("   ✅ Protocol: %s\n", resp.Protocol)
	fmt.Printf("   ✅ Timing: DNS=%v, TCP=%v, Total=%v\n",
		resp.Timing.DNSLookup, resp.Timing.TCPConnect, resp.Timing.Total)
}

func testHTTPS() {
	sender := rawhttp.NewSender()
	defer sender.Close()

	rawRequest := []byte("GET / HTTP/1.1\r\nHost: www.google.com\r\nConnection: close\r\n\r\n")

	opts := rawhttp.Options{
		Scheme: "https",
		Host:   "www.google.com",
		Port:   443,
	}

	ctx := context.Background()
	resp, err := sender.Do(ctx, rawRequest, opts)
	if err != nil {
		log.Printf("   ❌ HATA: %v\n", err)
		return
	}

	fmt.Printf("   ✅ Status Code: %d\n", resp.StatusCode)
	fmt.Printf("   ✅ TLS Handshake: %v\n", resp.Timing.TLSHandshake)
	fmt.Printf("   ✅ Protocol: %s\n", resp.Protocol)
	fmt.Printf("   ✅ Raw Response Length: %d bytes\n", len(resp.Raw))
}

func testConnectionReuse() {
	sender := rawhttp.NewSender()
	defer sender.Close()

	rawRequest := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")

	opts := rawhttp.Options{
		Scheme:          "http",
		Host:            "example.com",
		Port:            80,
		ReuseConnection: true, // Enable connection pooling
	}

	ctx := context.Background()
	var totalTime time.Duration

	for i := 0; i < 3; i++ {
		start := time.Now()
		resp, err := sender.Do(ctx, rawRequest, opts)
		elapsed := time.Since(start)
		totalTime += elapsed

		if err != nil {
			log.Printf("   ❌ Request %d HATA: %v\n", i+1, err)
			continue
		}

		fmt.Printf("   ✅ Request %d: Status=%d, Time=%v\n", i+1, resp.StatusCode, elapsed)
	}

	fmt.Printf("   ✅ Total time for 3 requests: %v\n", totalTime)
}

func testTimeouts() {
	sender := rawhttp.NewSender()
	defer sender.Close()

	rawRequest := []byte("GET /delay/10 HTTP/1.1\r\nHost: httpbin.org\r\nConnection: close\r\n\r\n")

	opts := rawhttp.Options{
		Scheme:      "http",
		Host:        "httpbin.org",
		Port:        80,
		ReadTimeout: 2 * time.Second, // Short timeout
	}

	ctx := context.Background()
	_, err := sender.Do(ctx, rawRequest, opts)

	if err != nil {
		fmt.Printf("   ✅ Timeout beklendiği gibi çalıştı: %v\n", err)
	} else {
		fmt.Printf("   ⚠️  Timeout beklenmiyordu ama response alındı\n")
	}
}

func testRawResponsePreservation() {
	sender := rawhttp.NewSender()
	defer sender.Close()

	rawRequest := []byte("GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n")

	opts := rawhttp.Options{
		Scheme: "http",
		Host:   "example.com",
		Port:   80,
	}

	ctx := context.Background()
	resp, err := sender.Do(ctx, rawRequest, opts)
	if err != nil {
		log.Printf("   ❌ HATA: %v\n", err)
		return
	}

	// Raw response'un başlangıcını göster
	rawPreview := string(resp.Raw)
	if len(rawPreview) > 200 {
		rawPreview = rawPreview[:200] + "..."
	}

	fmt.Printf("   ✅ Raw Response Preview:\n")
	fmt.Printf("   %s\n", rawPreview)
	fmt.Printf("   ✅ Total Raw Size: %d bytes\n", len(resp.Raw))
}

func testConnectionMetadata() {
	sender := rawhttp.NewSender()
	defer sender.Close()

	rawRequest := []byte("GET / HTTP/1.1\r\nHost: www.cloudflare.com\r\nConnection: close\r\n\r\n")

	opts := rawhttp.Options{
		Scheme: "https",
		Host:   "www.cloudflare.com",
		Port:   443,
	}

	ctx := context.Background()
	resp, err := sender.Do(ctx, rawRequest, opts)
	if err != nil {
		log.Printf("   ❌ HATA: %v\n", err)
		return
	}

	fmt.Printf("   ✅ Connected IP: %s\n", resp.ConnectedIP)
	fmt.Printf("   ✅ Connected Port: %d\n", resp.ConnectedPort)
	fmt.Printf("   ✅ Protocol: %s\n", resp.Protocol)
	fmt.Printf("   ✅ Status Code: %d\n", resp.StatusCode)
}

func testInsecureSkipVerify() {
	sender := rawhttp.NewSender()
	defer sender.Close()

	rawRequest := []byte("GET / HTTP/1.1\r\nHost: self-signed.badssl.com\r\nConnection: close\r\n\r\n")

	// First try without InsecureSkipVerify (should fail)
	opts := rawhttp.Options{
		Scheme:             "https",
		Host:               "self-signed.badssl.com",
		Port:               443,
		InsecureSkipVerify: false,
	}

	ctx := context.Background()
	_, err := sender.Do(ctx, rawRequest, opts)
	if err != nil {
		fmt.Printf("   ✅ Without InsecureSkipVerify: Failed as expected (%v)\n", err)
	} else {
		fmt.Printf("   ⚠️  Without InsecureSkipVerify: Unexpectedly succeeded\n")
	}

	// Now try with InsecureSkipVerify (should succeed)
	opts.InsecureSkipVerify = true
	resp, err := sender.Do(ctx, rawRequest, opts)
	if err != nil {
		fmt.Printf("   ❌ With InsecureSkipVerify: Failed (%v)\n", err)
	} else {
		fmt.Printf("   ✅ With InsecureSkipVerify: Success (Status=%d)\n", resp.StatusCode)
	}
}

func testErrorHandling() {
	sender := rawhttp.NewSender()
	defer sender.Close()

	rawRequest := []byte("GET / HTTP/1.1\r\nHost: invalid-host-that-does-not-exist-12345.com\r\n\r\n")

	opts := rawhttp.Options{
		Scheme: "http",
		Host:   "invalid-host-that-does-not-exist-12345.com",
		Port:   80,
	}

	ctx := context.Background()
	_, err := sender.Do(ctx, rawRequest, opts)

	if err != nil {
		fmt.Printf("   ✅ DNS error caught: %v\n", err)

		// Check if it's an HTTPError
		if httpErr, ok := err.(*rawhttp.HTTPError); ok {
			fmt.Printf("   ✅ Error Type: %v\n", httpErr.Type)
		}
	} else {
		fmt.Printf("   ❌ Expected DNS error but got success\n")
	}
}
