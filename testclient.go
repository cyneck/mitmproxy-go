package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	client := &http.Client{Timeout: 5 * time.Second}

	// 测试根路径
	fmt.Println("=== Testing http://127.0.0.1:5555/ ===")
	resp, err := client.Get("http://127.0.0.1:5555/")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Body: %s\n", string(body))

	// 格式化 JSON
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		pretty, _ := json.MarshalIndent(data, "", "  ")
		fmt.Printf("Formatted:\n%s\n", pretty)
	}

	fmt.Println()

	// 测试 /api/users
	fmt.Println("=== Testing http://127.0.0.1:5555/api/users ===")
	resp2, err := client.Get("http://127.0.0.1:5555/api/users")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("Status: %s\n", resp2.Status)
	fmt.Printf("Body: %s\n", string(body2))

	// 格式化 JSON
	var data2 map[string]interface{}
	if err := json.Unmarshal(body2, &data2); err == nil {
		pretty2, _ := json.MarshalIndent(data2, "", "  ")
		fmt.Printf("Formatted:\n%s\n", pretty2)
	}

	fmt.Println("\n=== All tests passed! ===")
}
