package src

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeProject holds what the provider can see and set on a Jira project.
type fakeProject struct {
	Key, Name, Description, Type, Lead string
	Archived                           bool
	IssueTypeScheme                    string
	WorkflowScheme                     int32
	PriorityScheme                     int32
	PermissionScheme                   int32
}

// Schemes a new project gets from the project template unless told otherwise.
const (
	defaultIssueTypeScheme  = "10000"
	defaultWorkflowScheme   = 10100
	defaultPriorityScheme   = 10200
	defaultPermissionScheme = 10300
)

// fakeJira is an in-memory Jira Data Center that serves the endpoints the provider
// calls, with the archived-project behaviour observed on a live instance: an
// archived project hides its workflow, priority and permission schemes (403 "You
// cannot view this project"), refuses changes (403 "You cannot edit the
// configuration of this project") and still accepts an issue type scheme.
type fakeJira struct {
	t        *testing.T
	mu       sync.Mutex
	projects map[string]*fakeProject
	writes   []string // "METHOD path" of every request that tried to change something
	server   *httptest.Server
}

func newFakeJira(t *testing.T) *fakeJira {
	t.Helper()
	f := &fakeJira{t: t, projects: map[string]*fakeProject{}}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeJira) URL() string { return f.server.URL + "/" }

// put stores a project as if it already existed in Jira.
func (f *fakeJira) put(p fakeProject) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.projects[p.Key] = &p
}

// project returns a copy of a stored project, or nil.
func (f *fakeJira) project(key string) *fakeProject {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.projects[key]
	if !ok {
		return nil
	}
	c := *p
	return &c
}

// writeCount is a mark to compare against: writesSince(mark) lists what came after.
func (f *fakeJira) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.writes)
}

func (f *fakeJira) writesSince(mark int) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.writes[mark:]...)
}

var (
	cannotView = `{"errorMessages":["You cannot view this project."],"errors":{}}`
	cannotEdit = `{"errorMessages":["You cannot edit the configuration of this project."],"errors":{}}`
)

func (f *fakeJira) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r.Method != http.MethodGet {
		f.writes = append(f.writes, r.Method+" "+r.URL.Path)
	}

	path := strings.TrimPrefix(r.URL.Path, "/rest/")
	switch {
	case r.Method == http.MethodPost && path == "api/2/project":
		f.createProject(w, r)
	case strings.HasPrefix(path, "api/2/project/"):
		f.serveProject(w, r, strings.Split(strings.TrimPrefix(path, "api/2/project/"), "/"))
	case r.Method == http.MethodPost && strings.HasPrefix(path, "api/2/issuetypescheme/") &&
		strings.HasSuffix(path, "/associations"):
		f.associateIssueTypeScheme(w, r, strings.TrimSuffix(strings.TrimPrefix(path, "api/2/issuetypescheme/"), "/associations"))
	case r.Method == http.MethodGet && path == "scriptrunner/latest/custom/getIssueTypeScheme":
		p := f.projects[r.URL.Query().Get("projectKey")]
		if p == nil {
			http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"projectKey": p.Key, "schemeId": p.IssueTypeScheme})
	case r.Method == http.MethodPut && path == "scriptrunner/latest/custom/assignWorkflowScheme":
		f.assignWorkflowScheme(w, r)
	default:
		f.unexpected(w, r)
	}
}

func (f *fakeJira) serveProject(w http.ResponseWriter, r *http.Request, parts []string) {
	p := f.projects[parts[0]]
	if p == nil {
		http.Error(w, `{"errorMessages":["No project could be found."]}`, http.StatusNotFound)
		return
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"key": p.Key, "name": p.Name, "description": p.Description,
			"projectTypeKey": p.Type, "lead": map[string]string{"name": p.Lead}, "archived": p.Archived,
		})
	case len(parts) == 1 && r.Method == http.MethodPut:
		if p.Archived {
			http.Error(w, cannotEdit, http.StatusForbidden)
			return
		}
		var body map[string]any
		if !f.decode(w, r, &body) {
			return
		}
		for field, value := range body {
			switch field {
			case "name":
				p.Name = value.(string)
			case "description":
				p.Description = value.(string)
			case "lead":
				p.Lead = value.(string)
			case "projectTypeKey":
				p.Type = value.(string)
			case "permissionScheme":
				p.PermissionScheme = int32(value.(float64))
			}
		}
		w.WriteHeader(http.StatusOK)
	case len(parts) == 1 && r.Method == http.MethodDelete:
		delete(f.projects, p.Key)
		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 2 && r.Method == http.MethodPut && parts[1] == "archive":
		p.Archived = true
		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 2 && r.Method == http.MethodPut && parts[1] == "restore":
		p.Archived = false
		w.WriteHeader(http.StatusAccepted)
	case len(parts) == 3 && r.Method == http.MethodPut && parts[1] == "type":
		if p.Archived {
			http.Error(w, cannotEdit, http.StatusForbidden)
			return
		}
		p.Type = parts[2]
		w.WriteHeader(http.StatusOK)
	case len(parts) == 2 && r.Method == http.MethodGet &&
		(parts[1] == "workflowscheme" || parts[1] == "priorityscheme" || parts[1] == "permissionscheme"):
		if p.Archived {
			http.Error(w, cannotView, http.StatusForbidden)
			return
		}
		id := map[string]int32{
			"workflowscheme": p.WorkflowScheme, "priorityscheme": p.PriorityScheme, "permissionscheme": p.PermissionScheme,
		}[parts[1]]
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
	case len(parts) == 2 && r.Method == http.MethodPut && parts[1] == "priorityscheme":
		if p.Archived {
			http.Error(w, cannotEdit, http.StatusForbidden)
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		if !f.decode(w, r, &body) {
			return
		}
		id, err := strconv.Atoi(body.ID)
		if err != nil {
			http.Error(w, `{"error":"bad id"}`, http.StatusBadRequest)
			return
		}
		p.PriorityScheme = int32(id)
		w.WriteHeader(http.StatusOK)
	default:
		f.unexpected(w, r)
	}
}

func (f *fakeJira) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key, Name, Description, Lead string
		ProjectTypeKey               string `json:"projectTypeKey"`
		PermissionScheme             int32  `json:"permissionScheme"`
	}
	if !f.decode(w, r, &body) {
		return
	}
	if _, exists := f.projects[body.Key]; exists {
		http.Error(w, `{"errors":{"projectKey":"A project with that project key already exists."}}`, http.StatusBadRequest)
		return
	}
	p := &fakeProject{
		Key: body.Key, Name: body.Name, Description: body.Description, Type: body.ProjectTypeKey, Lead: body.Lead,
		IssueTypeScheme: defaultIssueTypeScheme, WorkflowScheme: defaultWorkflowScheme,
		PriorityScheme: defaultPriorityScheme, PermissionScheme: defaultPermissionScheme,
	}
	if body.PermissionScheme != 0 {
		p.PermissionScheme = body.PermissionScheme
	}
	f.projects[p.Key] = p
	writeJSON(w, http.StatusCreated, map[string]any{"key": p.Key, "id": 10000 + len(f.projects)})
}

func (f *fakeJira) associateIssueTypeScheme(w http.ResponseWriter, r *http.Request, schemeID string) {
	var body struct {
		IdsOrKeys []string `json:"idsOrKeys"`
	}
	if !f.decode(w, r, &body) {
		return
	}
	for _, key := range body.IdsOrKeys {
		p := f.projects[key]
		if p == nil {
			http.Error(w, `{"errorMessages":["No project could be found."]}`, http.StatusNotFound)
			return
		}
		p.IssueTypeScheme = schemeID // accepted even for an archived project
	}
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeJira) assignWorkflowScheme(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectKey string `json:"projectKey"`
		SchemeID   int32  `json:"schemeId"`
	}
	if !f.decode(w, r, &body) {
		return
	}
	p := f.projects[body.ProjectKey]
	switch {
	case p == nil:
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
	case p.Archived:
		http.Error(w, cannotEdit, http.StatusForbidden)
	default:
		p.WorkflowScheme = body.SchemeID
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "projectKey": p.Key, "schemeId": p.WorkflowScheme})
	}
}

func (f *fakeJira) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return false
	}
	return true
}

// unexpected fails the test on a call the fake does not model, so a new request in
// the provider cannot pass unnoticed. 418 rather than 5xx: the client retries 5xx.
func (f *fakeJira) unexpected(w http.ResponseWriter, r *http.Request) {
	f.t.Errorf("fake Jira: unexpected request %s %s", r.Method, r.URL)
	http.Error(w, `{"error":"not modelled by the fake"}`, http.StatusTeapot)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
