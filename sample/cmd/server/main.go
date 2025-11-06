package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/toumakido/my-claude/sample/internal/handler"
	"github.com/toumakido/my-claude/sample/internal/store"
)

func main() {
	// Create in-memory store
	memStore := store.NewMemoryStore()

	// Create handler
	todoHandler := handler.NewTodoHandler(memStore)

	// Setup routes
	http.Handle("/todos", todoHandler)
	http.Handle("/todos/", todoHandler)

	// Root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"Welcome to Todo API","endpoints":{"/todos":"GET, POST","/todos/:id":"GET, PUT, DELETE"}}`)
	})

	// Start server
	port := ":8080"
	log.Printf("Server starting on http://localhost%s", port)
	log.Printf("Try: curl http://localhost%s/", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
