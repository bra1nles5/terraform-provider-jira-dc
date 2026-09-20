package jira

import (
	"encoding/json"
	"strings"
)

// SchemeID is a scheme identifier that ScriptRunner endpoints serialise either as a
// JSON number or as a JSON string, depending on how the Groovy script builds it.
// Both forms decode into the same string.
type SchemeID string

func (s *SchemeID) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "null" {
		*s = ""
		return nil
	}

	var quoted string
	if err := json.Unmarshal(data, &quoted); err == nil {
		*s = SchemeID(quoted)
		return nil
	}

	*s = SchemeID(text)
	return nil
}

type Project struct {
	Key              string `json:"key"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Type             string `json:"type"`
	Lead             string `json:"lead"`
	Archived         bool   `json:"archived"`
	PermissionScheme int32  `json:"permissionScheme"`
	WorkflowSchemeId int32  `json:"workflowSchemeId"`
}

type CreateProjectData struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Lead           string `json:"lead"`
	ProjectTypeKey string `json:"projectTypeKey"`
	// omitempty: without a permission scheme Jira creates a project that not even an
	// administrator can see, and the very next call fails with "You must have the view
	// project permission". Sending zero is wrong, so send nothing at all.
	PermissionScheme int32 `json:"permissionScheme,omitempty"`
}

// IssueTypeAssociationRequest associates projects with an issue type scheme.
// ProjectKeys holds project keys/IDs (Jira API field: "idsOrKeys").
type IssueTypeAssociationRequest struct {
	ProjectKeys []string `json:"idsOrKeys"`
}

type PrioritySchemeAssociationRequests struct {
	ID string `json:"id"`
}

type WorkflowChangeRequest struct {
	ProjectKey  string `json:"projectKey"`
	SchemeId    int32  `json:"schemeId"`
	UpdateDraft bool   `json:"updateDraft"`
}

type permissionSchemeAssignRequest struct {
	PermissionScheme int32 `json:"permissionScheme"`
}

type ProjectService struct {
	Client *JiraClient
}
