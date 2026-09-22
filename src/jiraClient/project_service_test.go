package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/project/TEST" {
			t.Fatalf("unexpected URL: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"key":            "TEST",
			"name":           "Test Project",
			"description":    "A test project",
			"projectTypeKey": "software",
			"lead":           map[string]string{"name": "alice"},
			"archived":       false,
		})
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	prj, err := client.Projects.Get(context.Background(), "TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prj.Key != "TEST" {
		t.Errorf("expected key 'TEST', got '%s'", prj.Key)
	}
	if prj.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got '%s'", prj.Name)
	}
	if prj.Lead != "alice" {
		t.Errorf("expected lead 'alice', got '%s'", prj.Lead)
	}
	if prj.Type != "software" {
		t.Errorf("expected type 'software', got '%s'", prj.Type)
	}
}

func TestProjectService_Get_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	_, err := client.Projects.Get(context.Background(), "MISSING")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestProjectService_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/2/project" {
			t.Fatalf("unexpected URL: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"key":  "NEW",
			"name": "New Project",
		})
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	prj, err := client.Projects.Create(context.Background(), Project{
		Key:  "NEW",
		Name: "New Project",
		Type: "software",
		Lead: "bob",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prj.Key != "NEW" {
		t.Errorf("expected key 'NEW', got '%s'", prj.Key)
	}
}

func TestProjectService_Update(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/2/project/TEST" {
			t.Fatalf("unexpected URL: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	err := client.Projects.Update(context.Background(), &Project{Key: "TEST", Name: "Updated Name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectService_Remove(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/2/project/TEST" {
			t.Fatalf("unexpected URL: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	err := client.Projects.Remove(context.Background(), &Project{Key: "TEST"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// A refused DELETE must not look like success: Terraform would drop the project
// from state while it stays in Jira.
func TestProjectService_Remove_ErrorStatus(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(`{"errorMessages":["nope"]}`))
		}))

		client := NewJiraClient(server.URL+"/", "user", "token")
		err := client.Projects.Remove(context.Background(), &Project{Key: "TEST"})
		if err == nil {
			t.Errorf("status %d: expected an error, got nil", status)
		}
		server.Close()
	}
}

func TestProjectService_Remove_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	err := client.Projects.Remove(context.Background(), &Project{Key: "TEST"})
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}
