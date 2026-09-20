package jira

import (
	"context"
	"fmt"
	"sync"
)

type WorkflowTypeScheme struct {
	Self        string `json:"self"`
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type workflowSchemesResponse struct {
	Schemes []WorkflowTypeScheme `json:"schemes"`
}

type WorkflowTypeSchemeService struct {
	client *JiraClient
	mu     sync.RWMutex
	Cache  map[string]WorkflowTypeScheme
}

func (s *WorkflowTypeSchemeService) Get(ctx context.Context, id int32) (*WorkflowTypeScheme, error) {
	resp, err := s.client.doRequest(ctx, "GET", fmt.Sprintf("rest/api/2/workflowscheme/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var result WorkflowTypeScheme
	if err := s.client.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetAll indexes every workflow scheme by name. REST API v2 of Jira DC has no listing
// for workflow schemes, so this goes through the listWorkflowSchemes ScriptRunner
// endpoint (deployment is described in the README).
func (s *WorkflowTypeSchemeService) GetAll(ctx context.Context) (map[string]WorkflowTypeScheme, error) {
	s.mu.RLock()
	if s.Cache != nil {
		defer s.mu.RUnlock()
		return s.Cache, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Cache != nil {
		return s.Cache, nil
	}

	resp, err := s.client.doRequest(ctx, "GET", "rest/scriptrunner/latest/custom/listWorkflowSchemes", nil)
	if err != nil {
		return nil, err
	}

	var result workflowSchemesResponse
	if err := s.client.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	s.Cache = make(map[string]WorkflowTypeScheme, len(result.Schemes))
	for _, scheme := range result.Schemes {
		s.Cache[scheme.Name] = scheme
	}

	return s.Cache, nil
}

func (s *WorkflowTypeSchemeService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cache = nil
}
