package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"example.com/pz3-http/internal/storage"
)

type Handlers struct {
	Store *storage.MemoryStore
}

func (h *Handlers) UpdateTask(w *httptest.ResponseRecorder, req *http.Request) {
	panic("unimplemented")
}

func NewHandlers(store *storage.MemoryStore) *Handlers {
	return &Handlers{Store: store}
}

// GET /tasks
func (h *Handlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := h.Store.List()

	// Поддержка простых фильтров через query: ?q=text
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q != "" {
		filtered := make([]*storage.Task, 0, len(tasks))
		for _, t := range tasks {
			if strings.Contains(strings.ToLower(t.Title), strings.ToLower(q)) {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	JSON(w, http.StatusOK, tasks)
}

type createTaskRequest struct {
	Title string `json:"title"`
}

// POST /tasks
func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "" && !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		BadRequest(w, "Content-Type must be application/json")
		return
	}

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid json: "+err.Error())
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	// VALIDATION: title length 3..140
	if len(req.Title) < 3 || len(req.Title) > 140 {
		Unprocessable(w, "title length must be between 3 and 140 characters")
		return
	}

	t := h.Store.Create(req.Title)
	JSON(w, http.StatusCreated, t)
}

func Unprocessable(w http.ResponseWriter, s string) {
	panic("unimplemented")
}

// GET /tasks/{id}
func (h *Handlers) GetTask(w http.ResponseWriter, r *http.Request) {
	// Ожидаем путь вида /tasks/123
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 {
		NotFound(w, "invalid path")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		BadRequest(w, "invalid id")
		return
	}

	t, err := h.Store.Get(id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			NotFound(w, "task not found")
			return
		}
		Internal(w, "unexpected error")
		return
	}
	JSON(w, http.StatusOK, t)
}

// PATCH /tasks/{id}  -- ожидается JSON { "done": true }
type patchTaskRequest struct {
	Done *bool `json:"done"`
}

func (h *Handlers) PatchTask(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "" && !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		BadRequest(w, "Content-Type must be application/json")
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 {
		NotFound(w, "invalid path")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		BadRequest(w, "invalid id")
		return
	}

	var req patchTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid json: "+err.Error())
		return
	}
	if req.Done == nil {
		BadRequest(w, "missing field done")
		return
	}

	t, err := h.Store.UpdateDone(id, *req.Done)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			NotFound(w, "task not found")
			return
		}
		Internal(w, "unexpected error")
		return
	}

	JSON(w, http.StatusOK, t)
}

// DELETE /tasks/{id}
func (h *Handlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 {
		NotFound(w, "invalid path")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		BadRequest(w, "invalid id")
		return
	}

	if err := h.Store.Delete(id); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			NotFound(w, "task not found")
			return
		}
		Internal(w, "unexpected error")
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204
}
