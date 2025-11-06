package store

import (
	"errors"
	"sync"
	"time"

	"github.com/toumakido/my-claude/sample/internal/model"
)

var (
	ErrNotFound = errors.New("todo not found")
)

// MemoryStore is an in-memory implementation of todo storage
type MemoryStore struct {
	mu      sync.RWMutex
	todos   map[int]*model.Todo
	nextID  int
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		todos:  make(map[int]*model.Todo),
		nextID: 1,
	}
}

// GetAll returns all todos
func (s *MemoryStore) GetAll() []*model.Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := make([]*model.Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		todos = append(todos, todo)
	}
	return todos
}

// GetByID returns a todo by ID
func (s *MemoryStore) GetByID(id int) (*model.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todo, ok := s.todos[id]
	if !ok {
		return nil, ErrNotFound
	}
	return todo, nil
}

// Create creates a new todo
func (s *MemoryStore) Create(title string) *model.Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	todo := &model.Todo{
		ID:        s.nextID,
		Title:     title,
		Completed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.todos[s.nextID] = todo
	s.nextID++
	return todo
}

// Update updates an existing todo
func (s *MemoryStore) Update(id int, title string, completed bool) (*model.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.todos[id]
	if !ok {
		return nil, ErrNotFound
	}

	todo.Title = title
	todo.Completed = completed
	todo.UpdatedAt = time.Now()
	return todo, nil
}

// Delete deletes a todo by ID
func (s *MemoryStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.todos[id]; !ok {
		return ErrNotFound
	}
	delete(s.todos, id)
	return nil
}
