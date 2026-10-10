package src

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// configWith is the project from projectConfig's defaults with some attributes
// replaced, for example {"name": `"Renamed"`}.
func configWith(f *fakeJira, overrides map[string]string) string {
	attrs := map[string]string{
		"key": `"EX"`, "name": `"Example"`, "type": `"software"`, "lead": `"jsmith"`, "archived": "false",
		"issue_type_schema_id": `"11947"`, "workflow_schema_id": "11601",
		"priority_schema_id": "11907", "permission_schema_id": "10600",
	}
	for k, v := range overrides {
		attrs[k] = v
	}
	names := make([]string, 0, len(attrs))
	for k := range attrs {
		names = append(names, k)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString(providerBlock(f))
	b.WriteString("\nresource \"jira_project\" \"test\" {\n")
	for _, k := range names {
		fmt.Fprintf(&b, "  %s = %s\n", k, attrs[k])
	}
	b.WriteString("}\n")
	return b.String()
}

// request matches a request by method and the end of its path.
func request(method, pathSuffix string) func(*fakeJira, *http.Request) bool {
	return func(_ *fakeJira, r *http.Request) bool {
		return r.Method == method && strings.HasSuffix(r.URL.Path, pathSuffix)
	}
}

// TestProjectResource_JiraFailures makes Jira fail one call at a time. Every failure
// must surface as an error naming the step, and once Jira recovers the same
// configuration must apply and converge: nothing is left half-recorded.
func TestProjectResource_JiraFailures(t *testing.T) {
	archived := map[string]string{"archived": "true"}

	cases := []struct {
		name   string
		before map[string]string // applied first; nil means the failure hits creation
		config map[string]string // the configuration whose apply meets the failure
		fail   func(*fakeJira, *http.Request) bool
		err    string
	}{
		// Creation. A failing issue type assignment is retried for ten seconds by
		// design (see retryAfterCreate) and is left out to keep the suite fast.
		{"create: project", nil, nil, request(http.MethodPost, "/rest/api/2/project"), "Error creating project"},
		{"create: priority scheme", nil, nil, request(http.MethodPut, "/EX/priorityscheme"), "Error assigning priority schema"},
		{"create: permission scheme", nil, nil, request(http.MethodPut, "/project/EX"), "Error assigning permission schema"},
		{"create: workflow scheme", nil, nil, request(http.MethodPut, "/assignWorkflowScheme"), "Error assigning workflow"},
		{"create: reading schemes back", nil, nil, request(http.MethodGet, "/getIssueTypeScheme"), "Schemas did not survive project creation"},
		{"create: archiving", nil, archived, request(http.MethodPut, "/EX/archive"), "Error archiving project after create"},
		{"create: reading the project", nil, nil, request(http.MethodGet, "/project/EX"), "Error reading project after create"},

		// Refresh.
		{"read: project", map[string]string{}, nil, request(http.MethodGet, "/project/EX"), "Error reading project"},
		{"read: issue type scheme", map[string]string{}, nil, request(http.MethodGet, "/getIssueTypeScheme"), "Error reading issue type schema"},
		{"read: workflow scheme", map[string]string{}, nil, request(http.MethodGet, "/EX/workflowscheme"), "Error reading workflow schema"},
		{"read: priority scheme", map[string]string{}, nil, request(http.MethodGet, "/EX/priorityscheme"), "Error reading priority schema"},
		{"read: permission scheme", map[string]string{}, nil, request(http.MethodGet, "/EX/permissionscheme"), "Error reading permission schema"},

		// Update.
		{"update: base fields", map[string]string{}, map[string]string{"name": `"Renamed"`},
			request(http.MethodPut, "/project/EX"), "Error updating project"},
		{"update: type", map[string]string{}, map[string]string{"type": `"business"`},
			request(http.MethodPut, "/EX/type/business"), "Error updating project type"},
		{"update: issue type scheme", map[string]string{}, map[string]string{"issue_type_schema_id": `"12000"`},
			request(http.MethodPost, "/associations"), "Error updating issue type schema"},
		{"update: priority scheme", map[string]string{}, map[string]string{"priority_schema_id": "12001"},
			request(http.MethodPut, "/EX/priorityscheme"), "Error updating priority schema"},
		{"update: permission scheme", map[string]string{}, map[string]string{"permission_schema_id": "12002"},
			request(http.MethodPut, "/project/EX"), "Error updating permission schema"},
		{"update: workflow scheme", map[string]string{}, map[string]string{"workflow_schema_id": "12003"},
			request(http.MethodPut, "/assignWorkflowScheme"), "Error updating workflow"},
		{"update: archiving", map[string]string{}, archived,
			request(http.MethodPut, "/EX/archive"), "Error archiving project"},
		{"update: restoring", archived, nil,
			request(http.MethodPut, "/EX/restore"), "Error restoring project"},
		{"update: reading schemes after restoring", archived, nil,
			func(f *fakeJira, r *http.Request) bool {
				// Only once restored: while archived the read answers 403, which is expected.
				return r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/EX/workflowscheme") &&
					!f.projects["EX"].Archived
			}, "Error reading schemes of the restored project"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newFakeJira(t)
			config := configWith(f, tc.config)

			var steps []resource.TestStep
			if tc.before != nil {
				steps = append(steps, resource.TestStep{Config: configWith(f, tc.before)})
			}
			steps = append(steps,
				resource.TestStep{
					PreConfig:   func() { f.set(func(f *fakeJira) { f.failOn = tc.fail }) },
					Config:      config,
					ExpectError: regexp.MustCompile(tc.err),
				},
				resource.TestStep{
					// Jira works again: the same configuration applies and the plan converges.
					PreConfig: func() { f.set(func(f *fakeJira) { f.failOn = nil }) },
					Config:    config,
				},
			)

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: providerFactories(),
				CheckDestroy:             checkGone(f, "EX"),
				Steps:                    steps,
			})
		})
	}
}
