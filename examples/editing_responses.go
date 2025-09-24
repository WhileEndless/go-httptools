package main

import (
	"fmt"
	"log"
	
	"github.com/WhileEndless/go-httptools/pkg/response"
	"github.com/WhileEndless/go-httptools/pkg/utils"
)

func main() {
	// Parse original response
	rawResponse := []byte(`HTTP/1.1 404 Not Found
Content-Type: text/html
Content-Length: 13
Server: Apache/2.4.41

Page not found`)

	resp, err := response.Parse(rawResponse)
	if err != nil {
		log.Fatal("Failed to parse response:", err)
	}
	
	fmt.Println("=== Original Response ===")
	fmt.Println(resp.BuildString())
	
	// Edit response using ResponseEditor
	editor := utils.NewResponseEditor(resp)
	editedResp := editor.
		SetStatusCode(200).
		SetStatusText("OK").
		UpdateHeader("Content-Type", "application/json").
		AddHeader("Cache-Control", "no-cache").
		SetBodyString(`{"message":"Success","data":{"user_id":123}}`, false).
		GetResponse()
	
	fmt.Println("\n=== Edited Response ===")
	fmt.Println(editedResp.BuildString())
	
	// Example with compression
	fmt.Println("\n=== Response with Compression ===")
	compressedEditor := utils.NewResponseEditor(resp)
	compressedResp := compressedEditor.
		SetStatusCode(200).
		SetStatusText("OK").
		UpdateHeader("Content-Type", "application/json").
		AddHeader("Content-Encoding", "gzip").
		SetBodyString(`{"message":"Compressed response","data":{"items":[1,2,3,4,5]}}`, true).
		GetResponse()
	
	fmt.Printf("Compressed: %t\n", compressedResp.Compressed)
	fmt.Printf("Decompressed body: %s\n", string(compressedResp.Body))
	fmt.Printf("Raw body size: %d bytes\n", len(compressedResp.RawBody))
	
	// Build decompressed version
	fmt.Println("\n=== Decompressed Version ===")
	decompressed := compressedResp.BuildDecompressed()
	fmt.Println(string(decompressed))
}