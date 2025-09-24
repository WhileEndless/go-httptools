package main

import (
	"fmt"
	"log"
	
	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/utils"
)

func main() {
	// Parse original request
	rawRequest := []byte(`POST /api/login HTTP/1.1
Host: example.com
Content-Type: application/x-www-form-urlencoded
Content-Length: 23

username=admin&pass=123`)

	req, err := request.Parse(rawRequest)
	if err != nil {
		log.Fatal("Failed to parse request:", err)
	}
	
	fmt.Println("=== Original Request ===")
	fmt.Println(req.BuildString())
	
	// Edit request using RequestEditor (Burp Suite-like)
	editor := utils.NewRequestEditor(req)
	editedReq := editor.
		SetMethod("PUT").
		SetURL("/api/users/123").
		AddHeader("Authorization", "Bearer new-token").
		UpdateHeader("Content-Type", "application/json").
		SetBodyString(`{"username":"admin","password":"newpass123"}`).
		AddQueryParam("force", "true").
		GetRequest()
	
	fmt.Println("\n=== Edited Request ===")
	fmt.Println(editedReq.BuildString())
	
	// Validate the edited request
	validation := utils.ValidateRequest(editedReq)
	fmt.Println("\n=== Validation Results ===")
	fmt.Printf("Valid: %t\n", validation.Valid)
	if len(validation.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range validation.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}
	if len(validation.Errors) > 0 {
		fmt.Println("Errors:")
		for _, error := range validation.Errors {
			fmt.Printf("  - %s\n", error)
		}
	}
}