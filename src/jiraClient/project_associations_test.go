package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectService_GetIssueTypeSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/scriptrunner/latest/custom/getIssueTypeScheme" {
			t.Fatalf("unexpected URL path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("projectKey") != "TEST" {
			t.Fatalf("unexpected projectKey: %s", r.URL.Query().Get("projectKey"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"schemeId":   "10100",
			"schemeName": "Default Issue Type Scheme",
		})
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	id, err := client.Projects.GetIssueTypeSchema(context.Background(), "TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "10100" {
		t.Errorf("expected '10100', got '%s'", id)
	}
}

func TestProjectService_GetWorkflowSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/scriptrunner/latest/custom/getWorkflowScheme" {
			t.Fatalf("unexpected URL path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("projectKey") != "TEST" {
			t.Fatalf("unexpected projectKey: %s", r.URL.Query().Get("projectKey"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"schemeId": int32(20200),
		})
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	id, err := client.Projects.GetWorkflowSchema(context.Background(), "TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 20200 {
		t.Errorf("expected 20200, got %d", id)
	}
}

func TestProjectService_GetPrioritySchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/project/TEST/priorityscheme" {
			t.Fatalf("unexpected URL: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": int32(30300),
		})
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	id, err := client.Projects.GetPrioritySchema(context.Background(), "TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 30300 {
		t.Errorf("expected 30300, got %d", id)
	}
}

func TestProjectService_GetPermissionSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/project/TEST/permissionscheme" {
			t.Fatalf("unexpected URL: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": int32(40400),
		})
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	id, err := client.Projects.GetPermissionSchema(context.Background(), "TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 40400 {
		t.Errorf("expected 40400, got %d", id)
	}
}

// The Groovy endpoint builds schemeId from a Long, so Jira sends it as a JSON number.
// The provider must accept that as readily as the quoted form.
func TestProjectService_GetIssueTypeSchema_NumericID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"schemeId":   11755,
			"schemeName": "ABAC: Scrum Issue Type Scheme",
		})
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	id, err := client.Projects.GetIssueTypeSchema(context.Background(), "ABAC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "11755" {
		t.Errorf("expected '11755', got '%s'", id)
	}
}

// Archived projects answer 403 on their priority and permission schemes. The client
// must surface that as a typed error so the resource can tell it apart from a failure.
func TestProjectService_SchemaGetters_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errorMessages":["You cannot view this project."],"errors":{}}`))
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")

	if _, err := client.Projects.GetPrioritySchema(context.Background(), "ARCH"); !IsForbidden(err) {
		t.Errorf("priority: expected forbidden error, got %v", err)
	}
	if _, err := client.Projects.GetPermissionSchema(context.Background(), "ARCH"); !IsForbidden(err) {
		t.Errorf("permission: expected forbidden error, got %v", err)
	}
}

// Silently reporting success when Jira refuses is what made apply go green while
// the project kept someone else's scheme.
func TestProjectService_AssignIssueTypeSchema_ErrorStatus(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusInternalServerError} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(`{"errorMessages":["nope"]}`))
		}))

		client := NewJiraClient(server.URL+"/", "user", "token")
		err := client.Projects.AssignIssueTypeSchema(context.Background(), "EX", "10000")
		if err == nil {
			t.Errorf("status %d: expected an error, got nil", status)
		}
		server.Close()
	}
}

func TestProjectService_AssignWorkflow_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"scheme not found"}`))
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	if err := client.Projects.AssignWorkflow(context.Background(), "EX", 10001, true); err == nil {
		t.Error("expected an error, got nil")
	}
}

func TestProjectService_ArchiveRestore(t *testing.T) {
	tests := []struct {
		name   string
		call   func(*JiraClient) error
		path   string
		status int
	}{
		{"archive", func(c *JiraClient) error { return c.Projects.Archive(context.Background(), "TEST") },
			"/rest/api/2/project/TEST/archive", http.StatusNoContent},
		{"restore", func(c *JiraClient) error { return c.Projects.Restore(context.Background(), "TEST") },
			"/rest/api/2/project/TEST/restore", http.StatusAccepted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != tt.path {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			if err := tt.call(NewJiraClient(server.URL+"/", "user", "token")); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
