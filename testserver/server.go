package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	port := "8888"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	// 使用 127.0.0.1 只绑定本地回环接口
	addr := net.JoinHostPort("127.0.0.1", port)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Code:    0,
			Message: "Welcome to Test Server",
			Data: map[string]interface{}{
				"version":   "1.0.0",
				"endpoints": []string{"/api/users"},
			},
		})
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Code:    0,
			Message: "success",
			Data: []map[string]interface{}{
				{"id": 1, "name": "Alice", "email": "alice@example.com"},
				{"id": 2, "name": "Bob", "email": "bob@example.com"},
			},
		})
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	log.Printf("[TestServer] Starting on http://%s", addr)
	log.Printf("[TestServer] Endpoints:")
	log.Printf("  - GET http://localhost:%s/", port)
	log.Printf("  - GET http://localhost:%s/api/users", port)
	log.Printf("")

	if err := http.Serve(listener, mux); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
