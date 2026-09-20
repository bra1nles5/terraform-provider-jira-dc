package jira

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (s *ProjectService) GetProjectByKey(ctx context.Context) error {
	resp, err := s.Client.doRequest(ctx, "GET", "rest/api/2/project", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (s *ProjectService) Create(ctx context.Context, project Project) (*Project, error) {
	body := CreateProjectData{
		Key:              project.Key,
		Name:             project.Name,
		Description:      project.Description,
		Lead:             project.Lead,
		ProjectTypeKey:   project.Type,
		PermissionScheme: project.PermissionScheme,
	}
	tflog.Debug(ctx, "creating project", map[string]any{"key": body.Key, "name": body.Name})
	resp, err := s.Client.doRequest(ctx, "POST", "rest/api/2/project", body)
	if err != nil {
		return nil, err
	}
	var created Project
	err = s.Client.decodeJSON(resp, &created)
	return &created, err
}

func (s *ProjectService) Update(ctx context.Context, project *Project) error {
	body := map[string]interface{}{
		"name":           project.Name,
		"description":    project.Description,
		"lead":           project.Lead,
		"key":            project.Key,
		"projectTypeKey": project.Type,
	}
	tflog.Debug(ctx, "updating project", map[string]any{"key": project.Key})
	resp, err := s.Client.doRequest(
		ctx,
		"PUT",
		fmt.Sprintf("rest/api/2/project/%s", url.PathEscape(project.Key)),
		body,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to update project: %d", resp.StatusCode)
	}

	return nil
}

func (s *ProjectService) Get(ctx context.Context, projectKey string) (*Project, error) {
	resp, err := s.Client.doRequest(
		ctx,
		"GET",
		fmt.Sprintf("rest/api/2/project/%s", url.PathEscape(projectKey)),
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, &NotFoundError{StatusCode: 404}
	}

	var data struct {
		Key         string `json:"key"`
		Name        string `json:"name"`
		Description string `json:"description"`
		ProjectType string `json:"projectTypeKey"`
		Lead        struct {
			Name string `json:"name"`
		} `json:"lead"`
		Archived bool `json:"archived"`
	}

	err = s.Client.decodeJSON(resp, &data)
	if err != nil {
		return nil, err
	}

	return &Project{
		Key:         data.Key,
		Name:        data.Name,
		Description: data.Description,
		Type:        data.ProjectType,
		Lead:        data.Lead.Name,
		Archived:    data.Archived,
	}, nil
}

func (s *ProjectService) Remove(ctx context.Context, project *Project) error {
	tflog.Debug(ctx, "removing project", map[string]any{"key": project.Key})
	resp, err := s.Client.doRequest(ctx, "DELETE", fmt.Sprintf("rest/api/2/project/%s", url.PathEscape(project.Key)), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
