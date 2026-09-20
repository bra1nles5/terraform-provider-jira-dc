package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// NotFoundError is returned by API calls when the resource does not exist (HTTP 404).
type NotFoundError struct {
	StatusCode int
	Body       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("jira: not found (status %d): %s", e.StatusCode, e.Body)
}

// ForbiddenError is returned when Jira refuses to expose a resource (HTTP 403).
// Archived projects answer this way for their priority and permission schemes:
// "You cannot view this project".
type ForbiddenError struct {
	StatusCode int
	Body       string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("jira: forbidden (status %d): %s", e.StatusCode, e.Body)
}

type JiraClient struct {
	BaseURL  string
	Username string
	Token    string
	client   *http.Client

	IssueTypes  *IssueTypeSchemeService
	Priorities  *IssuePrioritySchemaService
	Workflow    *WorkflowTypeSchemeService
	Projects    *ProjectService
	Permissions *PermissionSchemaService
}

func NewJiraClient(baseURL, username, token string) *JiraClient {
	c := &JiraClient{
		BaseURL:  strings.TrimRight(baseURL, "/") + "/",
		Username: username,
		Token:    token,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
	c.IssueTypes = &IssueTypeSchemeService{client: c}
	c.Priorities = &IssuePrioritySchemaService{client: c}
	c.Workflow = &WorkflowTypeSchemeService{client: c}
	c.Projects = &ProjectService{Client: c}
	c.Permissions = &PermissionSchemaService{client: c}
	return c
}

// doRequest executes an HTTP request and retries up to 3 times on 429 or 5xx responses.
// On 429 it honours the Retry-After header; on 5xx it uses exponential back-off (1s/2s/4s).
func (c *JiraClient) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	tflog.Debug(ctx, "Jira API request", map[string]any{"method": method, "url": url})

	const maxRetries = 3
	for attempt := 0; ; attempt++ {
		var bodyReader io.Reader
		if body != nil {
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewBuffer(jsonData)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.Username, c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}

		if attempt < maxRetries && (resp.StatusCode == 429 || resp.StatusCode >= 500) {
			delay := retryDelay(attempt, resp)
			tflog.Debug(ctx, "Jira API retrying request", map[string]any{
				"attempt":  attempt + 1,
				"status":   resp.StatusCode,
				"delay_ms": delay.Milliseconds(),
			})
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		return resp, nil
	}
}

// retryDelay returns how long to wait before the next attempt.
// For 429 it reads Retry-After (seconds); for 5xx it uses exponential back-off.
func retryDelay(attempt int, resp *http.Response) time.Duration {
	if resp.StatusCode == 429 {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				return time.Duration(secs) * time.Second
			}
		}
		return 5 * time.Second
	}
	return time.Duration(1<<uint(attempt)) * time.Second
}

func (c *JiraClient) decodeJSON(resp *http.Response, v any) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira error %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func IsNotFound(err error) bool {
	var nfe *NotFoundError
	return errors.As(err, &nfe)
}

func IsForbidden(err error) bool {
	var fe *ForbiddenError
	return errors.As(err, &fe)
}
