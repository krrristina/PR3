package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"example.com/pz3-http/internal/storage"
)

func newTestServer() (*Handlers, *storage.MemoryStore, *httptest.Server) {
	store := storage.NewMemoryStore()
	h := NewHandlers(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListTasks(w, r)
		case http.MethodPost:
			h.CreateTask(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetTask(w, r)
		case http.MethodPatch:
			h.PatchTask(w, r)
		case http.MethodDelete:
			h.DeleteTask(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	ts := httptest.NewServer(mux)
	return h, store, ts
}

func TestCreateGetPatchDeleteTask(t *testing.T) {
	_, _, ts := newTestServer()
	defer ts.Close()

	// Create
	payload := `{"title":"Test task"}`
	res, err := http.Post(ts.URL+"/tasks", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var created map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatalf("decode create body: %v", err)
	}
	idf, ok := created["id"].(float64)
	if !ok {
		t.Fatalf("invalid id type")
	}
	id := int64(idf)

	// Get
	res, err = http.Get(ts.URL + "/tasks/" + strconvFormat(id))
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var got map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode get body: %v", err)
	}
	if gotTitle, _ := got["title"].(string); gotTitle != "Test task" {
		t.Fatalf("unexpected title: %v", gotTitle)
	}

	// Patch done = true
	patch := `{"done": true}`
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/tasks/"+strconvFormat(id), bytes.NewBufferString(patch))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("patch error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on patch, got %d", res.StatusCode)
	}
	var patched map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&patched); err != nil {
		t.Fatalf("decode patch body: %v", err)
	}
	if done, _ := patched["done"].(bool); !done {
		t.Fatalf("expected done true")
	}

	// Delete
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/tasks/"+strconvFormat(id), nil)
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 on delete, got %d", res.StatusCode)
	}

	// Get after delete -> 404
	res, err = http.Get(ts.URL + "/tasks/" + strconvFormat(id))
	if err != nil {
		t.Fatalf("get2 error: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", res.StatusCode)
	}
}

func TestCreateValidation(t *testing.T) {
	_, _, ts := newTestServer()
	defer ts.Close()

	// title too short (2 chars)
	payload := `{"title":"ab"}`
	res, err := http.Post(ts.URL+"/tasks", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for short title, got %d", res.StatusCode)
	}
}

// helper to avoid importing strconv many times in tests
func strconvFormat(i int64) string {
	return strconv.FormatInt(i, 10)
}
