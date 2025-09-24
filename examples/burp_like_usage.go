package main

import (
	"fmt"
	"log"
	"strings"
	
	"github.com/WhileEndless/go-httptools/pkg/request"
)

func main() {
	fmt.Println("=== Burp Suite-like Header Management ===\n")
	
	// Typical web request
	original := []byte(`POST /login HTTP/1.1
Host: target.com
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)
test:deneme
Content-Type: application/x-www-form-urlencoded
Content-Length: 27

username=admin&password=123`)

	fmt.Println("📋 Original Request:")
	rawReq, err := request.ParseRaw(original)
	if err != nil {
		log.Fatal(err)
	}
	
	for i, header := range rawReq.Headers.All() {
		fmt.Printf("  %d. %s: %s\n", i+1, header.Name, header.Value)
	}
	
	fmt.Println("\n🎯 Your Question: Auth header'ı Host'tan hemen sonra eklemek")
	fmt.Println("💡 Solution: SetAfter() kullan")
	
	// Host'tan hemen sonra Authorization ekle
	rawReq.Headers.SetAfter("Authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9", "Host")
	
	fmt.Println("\n✅ After SetAfter(\"Authorization\", \"Bearer ...\", \"Host\"):")
	for i, header := range rawReq.Headers.All() {
		if header.Name == "Authorization" {
			fmt.Printf("  %d. %s: %s ← 🎯 Host'tan hemen sonra!\n", i+1, header.Name, header.Value)
		} else {
			fmt.Printf("  %d. %s: %s\n", i+1, header.Name, header.Value)
		}
	}
	
	fmt.Println("\n📤 Rebuilt Request:")
	rebuilt := rawReq.BuildRawString()
	fmt.Println(rebuilt)
	
	fmt.Println(strings.Repeat("=", 70))
	
	// More examples
	fmt.Println("\n🔧 More Positioning Examples:\n")
	
	// 1. Cookie'yi başa ekle
	fmt.Println("1️⃣ Cookie'yi en başa ekle:")
	rawReq.Headers.SetAt("Cookie", "sessionid=abc123def456", 0)
	
	// 2. X-Forwarded-For'u User-Agent'tan önce ekle  
	fmt.Println("2️⃣ X-Forwarded-For'u User-Agent'tan önce ekle:")
	rawReq.Headers.SetBefore("X-Forwarded-For", "127.0.0.1", "User-Agent")
	
	// 3. Custom header'ı Content-Type'dan sonra ekle
	fmt.Println("3️⃣ X-API-Key'i Content-Type'dan sonra ekle:")
	rawReq.Headers.SetAfter("X-API-Key", "secret-api-key-12345", "Content-Type")
	
	fmt.Println("\n✅ Final Header Order:")
	for i, header := range rawReq.Headers.All() {
		symbol := ""
		switch header.Name {
		case "Cookie":
			symbol = " ← 🥪 En başta"
		case "Authorization":
			symbol = " ← 🔐 Host'tan sonra"
		case "X-Forwarded-For":
			symbol = " ← 🌐 User-Agent'tan önce"
		case "test":
			symbol = " ← ✨ Custom header korundu"
		case "X-API-Key":
			symbol = " ← 🔑 Content-Type'dan sonra"
		}
		fmt.Printf("  %d. %s: %s%s\n", i+1, header.Name, header.Value, symbol)
	}
	
	fmt.Println("\n📤 Final Request:")
	final := rawReq.BuildRawString()
	fmt.Println(final)
	
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("\n📚 Available Methods:")
	fmt.Println("• rawReq.Headers.Set(name, value)                    // Sona ekler")
	fmt.Println("• rawReq.Headers.SetAfter(name, value, afterHeader)  // Belirtilen header'dan sonra")  
	fmt.Println("• rawReq.Headers.SetBefore(name, value, beforeHeader)// Belirtilen header'dan önce")
	fmt.Println("• rawReq.Headers.SetAt(name, value, index)          // Belirli index'e")
	
	fmt.Println("\n✅ Custom 'test:deneme' header hala erişilebilir:", rawReq.Headers.Get("test"))
}