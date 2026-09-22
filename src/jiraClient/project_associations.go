package jira

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// forbidden converts a 403 response into a typed error and drains the body, so the
// caller can tell "Jira will not show this" apart from a genuine failure.
func forbidden(resp *http.Response) error {
	if resp.StatusCode != http.StatusForbidden {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return &ForbiddenError{StatusCode: resp.StatusCode, Body: string(body)}
}

// expectSuccess drains the body and turns any non-2xx answer into an error. Without
// it a write silently "succeeds" on 400 or 403, Terraform records the desired value
// and the drift surfaces only much later.
func expectSuccess(resp *http.Response, action string) error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusNotFound:
		return &NotFoundError{StatusCode: resp.StatusCode, Body: string(body)}
	case resp.StatusCode == http.StatusForbidden:
		return &ForbiddenError{StatusCode: resp.StatusCode, Body: string(body)}
	default:
		return fmt.Errorf("jira: %s failed (status %d): %s", action, resp.StatusCode, string(body))
	}
}

func (s *ProjectService) GetIssueTypeSchema(ctx context.Context, projectKey string) (string, error) {
	resp, err := s.Client.doRequest(
		ctx,
		"GET",
		fmt.Sprintf("rest/scriptrunner/latest/custom/getIssueTypeScheme?projectKey=%s", url.QueryEscape(projectKey)),
		nil,
	)
	if err != nil {
		return "", err
	}

	var data struct {
		ID               SchemeID `json:"schemeId"`
		PKey             string   `json:"projectKey"`
		Name             string   `json:"schemeName"`
		Description      string   `json:"schemeDescription"`
		DefaultIssueType string   `json:"defaultIssueType"`
	}
	if err := s.Client.decodeJSON(resp, &data); err != nil {
		return "", err
	}
	return string(data.ID), nil
}

func (s *ProjectService) GetWorkflowSchema(ctx context.Context, projectKey string) (int32, error) {
	resp, err := s.Client.doRequest(
		ctx,
		"GET",
		fmt.Sprintf("rest/api/2/project/%s/workflowscheme", url.PathEscape(projectKey)),
		nil,
	)
	if err != nil {
		return 0, err
	}

	if err := forbidden(resp); err != nil {
		return 0, err
	}

	var data struct {
		ID int32 `json:"id"`
	}

	if err := s.Client.decodeJSON(resp, &data); err != nil {
		return 0, err
	}

	return data.ID, nil
}

func (s *ProjectService) GetPrioritySchema(ctx context.Context, projectKey string) (int32, error) {
	resp, err := s.Client.doRequest(
		ctx,
		"GET",
		fmt.Sprintf("rest/api/2/project/%s/priorityscheme", url.PathEscape(projectKey)),
		nil,
	)
	if err != nil {
		return 0, err
	}

	if err := forbidden(resp); err != nil {
		return 0, err
	}

	var data struct {
		ID int32 `json:"id"`
	}

	if err := s.Client.decodeJSON(resp, &data); err != nil {
		return 0, err
	}

	return data.ID, nil
}

func (s *ProjectService) GetPermissionSchema(ctx context.Context, projectKey string) (int32, error) {
	resp, err := s.Client.doRequest(
		ctx,
		"GET",
		fmt.Sprintf("rest/api/2/project/%s/permissionscheme", url.PathEscape(projectKey)),
		nil,
	)
	if err != nil {
		return 0, err
	}

	if err := forbidden(resp); err != nil {
		return 0, err
	}

	var data struct {
		ID int32 `json:"id"`
	}

	if err := s.Client.decodeJSON(resp, &data); err != nil {
		return 0, err
	}

	return data.ID, nil
}

func (s *ProjectService) AssignIssueTypeSchema(ctx context.Context, ProjectKey string, issueTypeSchemaID string) error {
	tflog.Debug(ctx, "assigning issue type schema", map[string]any{"project_key": ProjectKey, "schema_id": issueTypeSchemaID})
	body := IssueTypeAssociationRequest{
		ProjectKeys: []string{ProjectKey},
	}
	resp, err := s.Client.doRequest(ctx, "POST", fmt.Sprintf("rest/api/2/issuetypescheme/%s/associations", issueTypeSchemaID), body)
	if err != nil {
		return err
	}
	return expectSuccess(resp, "assigning issue type scheme")
}

func (s *ProjectService) AssignPrioritySchema(ctx context.Context, ProjectKey string, PrioritySchemaID string) error {
	tflog.Debug(ctx, "assigning priority schema", map[string]any{"project_key": ProjectKey, "schema_id": PrioritySchemaID})
	body := PrioritySchemeAssociationRequests{
		ID: PrioritySchemaID,
	}
	resp, err := s.Client.doRequest(ctx, "PUT", fmt.Sprintf("rest/api/2/project/%s/priorityscheme", url.PathEscape(ProjectKey)), body)
	if err != nil {
		return err
	}
	return expectSuccess(resp, "assigning priority scheme")
}

func (s *ProjectService) UpdateProjectType(ctx context.Context, ProjectKey string, ProjectType string) error {
	tflog.Debug(ctx, "updating project type", map[string]any{"project_key": ProjectKey, "type": ProjectType})
	resp, err := s.Client.doRequest(ctx, "PUT", fmt.Sprintf("rest/api/2/project/%s/type/%s", url.PathEscape(ProjectKey), url.PathEscape(ProjectType)), nil)
	if err != nil {
		return err
	}
	return expectSuccess(resp, "updating project type")
}

func (s *ProjectService) AssignPermissionSchema(ctx context.Context, prj *Project, NewPermissionSchema int32) error {
	tflog.Debug(ctx, "assigning permission schema", map[string]any{"project_key": prj.Key, "schema_id": NewPermissionSchema})
	body := permissionSchemeAssignRequest{
		PermissionScheme: NewPermissionSchema,
	}
	resp, err := s.Client.doRequest(ctx, "PUT", fmt.Sprintf("rest/api/2/project/%s", url.PathEscape(prj.Key)), body)
	if err != nil {
		return err
	}
	return expectSuccess(resp, "assigning permission scheme")
}

func (s *ProjectService) AssignWorkflow(ctx context.Context, projectKey string, NewWorkflowSchemaId int32, updateDraft bool) error {
	tflog.Debug(ctx, "assigning workflow schema", map[string]any{"project_key": projectKey, "schema_id": NewWorkflowSchemaId})
	body := WorkflowChangeRequest{
		ProjectKey:  projectKey,
		SchemeId:    NewWorkflowSchemaId,
		UpdateDraft: updateDraft,
	}
	resp, err := s.Client.doRequest(ctx, "PUT", "rest/scriptrunner/latest/custom/assignWorkflowScheme", body)
	if err != nil {
		return err
	}
	return expectSuccess(resp, "assigning workflow scheme")
}

func (s *ProjectService) Archive(ctx context.Context, projectKey string) error {
	tflog.Debug(ctx, "archiving project", map[string]any{"key": projectKey})
	resp, err := s.Client.doRequest(ctx, "PUT", fmt.Sprintf("rest/api/2/project/%s/archive", url.PathEscape(projectKey)), nil)
	if err != nil {
		return err
	}
	return expectSuccess(resp, "archiving project")
}

func (s *ProjectService) Restore(ctx context.Context, projectKey string) error {
	tflog.Debug(ctx, "restoring project", map[string]any{"key": projectKey})
	resp, err := s.Client.doRequest(ctx, "PUT", fmt.Sprintf("rest/api/2/project/%s/restore", url.PathEscape(projectKey)), nil)
	if err != nil {
		return err
	}
	return expectSuccess(resp, "restoring project")
}
