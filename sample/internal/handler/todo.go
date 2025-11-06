package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/toumakido/my-claude/sample/internal/store"
)

// TodoHandler handles HTTP requests for todos
type TodoHandler struct {
	store *store.MemoryStore
}

// NewTodoHandler creates a new TodoHandler
func NewTodoHandler(store *store.MemoryStore) *TodoHandler {
	return &TodoHandler{store: store}
}

// ServeHTTP implements http.Handler
func (h *TodoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for local development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/todos")

	// GET /todos - Get all todos
	if r.Method == http.MethodGet && path == "" {
		h.handleGetAll(w, r)
		return
	}

	// GET /todos/:id - Get a specific todo
	if r.Method == http.MethodGet && path != "" {
		h.handleGetByID(w, r, path)
		return
	}

	// POST /todos - Create a new todo
	if r.Method == http.MethodPost && path == "" {
		h.handleCreate(w, r)
		return
	}

	// PUT /todos/:id - Update a todo
	if r.Method == http.MethodPut && path != "" {
		h.handleUpdate(w, r, path)
		return
	}

	// DELETE /todos/:id - Delete a todo
	if r.Method == http.MethodDelete && path != "" {
		h.handleDelete(w, r, path)
		return
	}

	http.NotFound(w, r)
}

func (h *TodoHandler) handleGetAll(w http.ResponseWriter, r *http.Request) {
	todos := h.store.GetAll()
	respondJSON(w, http.StatusOK, todos)
}

func (h *TodoHandler) handleGetByID(w http.ResponseWriter, r *http.Request, path string) {
	id, err := parseID(path)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	todo, err := h.store.GetByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondJSON(w, http.StatusOK, todo)
}

func (h *TodoHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "Title is required")
		return
	}

	todo := h.store.Create(req.Title)
	respondJSON(w, http.StatusCreated, todo)
}

func (h *TodoHandler) handleUpdate(w http.ResponseWriter, r *http.Request, path string) {
	id, err := parseID(path)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "Title is required")
		return
	}

	todo, err := h.store.Update(id, req.Title, req.Completed)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondJSON(w, http.StatusOK, todo)
}

func (h *TodoHandler) handleDelete(w http.ResponseWriter, r *http.Request, path string) {
	id, err := parseID(path)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func parseID(path string) (int, error) {
	idStr := strings.TrimPrefix(path, "/")
	return strconv.Atoi(idStr)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
