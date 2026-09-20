package jira

import (
	"context"
	"sync"
)

type IssuePrioritySchema struct {
	Self        string   `json:"self"`
	ID          int32    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	OptionIds   []string `json:"optionIds"`
}

type issuePrioritySchemaResponse struct {
	Schemes []IssuePrioritySchema `json:"schemes"`
}

type IssuePrioritySchemaService struct {
	client *JiraClient
	mu     sync.RWMutex
	Cache  map[string]IssuePrioritySchema
}

func (s *IssuePrioritySchemaService) GetAll(ctx context.Context) (map[string]IssuePrioritySchema, error) {
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

	resp, err := s.client.doRequest(ctx, "GET", "rest/api/2/priorityschemes", nil)
	if err != nil {
		return nil, err
	}

	var result issuePrioritySchemaResponse
	if err := s.client.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	s.Cache = make(map[string]IssuePrioritySchema, len(result.Schemes))
	for _, scheme := range result.Schemes {
		s.Cache[scheme.Name] = scheme
	}

	return s.Cache, nil
}

func (s *IssuePrioritySchemaService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cache = nil
}
