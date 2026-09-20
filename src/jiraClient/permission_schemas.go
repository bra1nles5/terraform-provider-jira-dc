package jira

import (
	"context"
	"sync"
)

type PermissionSchema struct {
	Self        string `json:"self"`
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PermissionSchemaResponse struct {
	Schemes []PermissionSchema `json:"permissionSchemes"`
}

type PermissionSchemaService struct {
	client *JiraClient
	mu     sync.RWMutex
	Cache  map[string]PermissionSchema
}

func (s *PermissionSchemaService) GetAll(ctx context.Context) (map[string]PermissionSchema, error) {
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

	resp, err := s.client.doRequest(ctx, "GET", "rest/api/2/permissionscheme", nil)
	if err != nil {
		return nil, err
	}

	var result PermissionSchemaResponse
	if err := s.client.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	s.Cache = make(map[string]PermissionSchema, len(result.Schemes))
	for _, scheme := range result.Schemes {
		s.Cache[scheme.Name] = scheme
	}

	return s.Cache, nil
}

func (s *PermissionSchemaService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cache = nil
}
