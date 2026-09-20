package jira

import (
	"context"
	"sync"
)

type IssueTypeScheme struct {
	Self        string `json:"self"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type issueTypeSchemesResponse struct {
	Schemes []IssueTypeScheme `json:"schemes"`
}

type IssueTypeSchemeService struct {
	client *JiraClient
	mu     sync.RWMutex
	Cache  map[string]IssueTypeScheme
}

func (s *IssueTypeSchemeService) GetAll(ctx context.Context) (map[string]IssueTypeScheme, error) {
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

	resp, err := s.client.doRequest(ctx, "GET", "rest/api/2/issuetypescheme", nil)
	if err != nil {
		return nil, err
	}

	var result issueTypeSchemesResponse
	if err := s.client.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	s.Cache = make(map[string]IssueTypeScheme, len(result.Schemes))
	for _, scheme := range result.Schemes {
		s.Cache[scheme.Name] = scheme
	}

	return s.Cache, nil
}

func (s *IssueTypeSchemeService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cache = nil
}
