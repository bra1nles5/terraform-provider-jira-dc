package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIssueTypeSchemeService_GetAll(t *testing.T) {
	mockResponse := issueTypeSchemesResponse{
		Schemes: []IssueTypeScheme{
			{ID: "10001", Name: "Default Issue Type Scheme", Description: "Default scheme"},
			{ID: "10002", Name: "Scrum Issue Type Scheme", Description: "Scrum scheme"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/issuetypescheme" {
			t.Fatalf("unexpected URL: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")

	cache, err := client.IssueTypes.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cache) != 2 {
		t.Fatalf("expected 2 schemes, got %d", len(cache))
	}

	scheme, ok := cache["Default Issue Type Scheme"]
	if !ok {
		t.Fatal("expected 'Default Issue Type Scheme' in cache")
	}
	if scheme.ID != "10001" {
		t.Errorf("expected ID '10001', got '%s'", scheme.ID)
	}
}

func TestIssueTypeSchemeService_GetAll_CachesResult(t *testing.T) {
	callCount := 0
	mockResponse := issueTypeSchemesResponse{
		Schemes: []IssueTypeScheme{
			{ID: "10001", Name: "Default Issue Type Scheme"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	ctx := context.Background()

	client.IssueTypes.GetAll(ctx)
	client.IssueTypes.GetAll(ctx)
	client.IssueTypes.GetAll(ctx)

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call due to caching, got %d", callCount)
	}
}

func TestIssueTypeSchemeService_ClearCache(t *testing.T) {
	callCount := 0
	mockResponse := issueTypeSchemesResponse{
		Schemes: []IssueTypeScheme{
			{ID: "10001", Name: "Default Issue Type Scheme"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")
	ctx := context.Background()

	client.IssueTypes.GetAll(ctx)
	client.IssueTypes.ClearCache()
	client.IssueTypes.GetAll(ctx)

	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls after ClearCache, got %d", callCount)
	}
}

func TestWorkflowTypeSchemeService_GetAll(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/scriptrunner/latest/custom/listWorkflowSchemes" {
			t.Fatalf("unexpected URL path: %s", r.URL.Path)
		}
		calls++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"schemes": []map[string]any{
				{"id": 14003, "name": "PRJ: Workflow Scheme"},
				{"id": 10501, "name": "Default Workflow Scheme"},
			},
		})
	}))
	defer server.Close()

	client := NewJiraClient(server.URL+"/", "user", "token")

	cache, err := client.Workflow.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cache["PRJ: Workflow Scheme"].ID; got != 14003 {
		t.Errorf("expected id 14003, got %d", got)
	}

	if _, err := client.Workflow.GetAll(context.Background()); err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected the listing to be fetched once, got %d calls", calls)
	}

	client.Workflow.ClearCache()
	if _, err := client.Workflow.GetAll(context.Background()); err != nil {
		t.Fatalf("unexpected error after ClearCache: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected a refetch after ClearCache, got %d calls", calls)
	}
}
