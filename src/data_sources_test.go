package src

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// withSchemes fills the fake with the schemes a default Jira instance has.
func withSchemes(f *fakeJira) {
	f.set(func(f *fakeJira) {
		f.issueTypeSchemes = []fakeScheme{{ID: 11947, Name: "Default Issue Type Scheme", Description: "Default for new projects"}}
		f.prioritySchemes = []fakeScheme{{ID: 11907, Name: "Default Priority Scheme", Description: "Standard priorities", OptionIDs: []string{"1", "2", "3"}}}
		f.workflowSchemes = []fakeScheme{{ID: 11601, Name: "Default Workflow Scheme", Description: "Software workflow"}}
		f.permissionSchemes = []fakeScheme{{ID: 10600, Name: "Default Permission Scheme", Description: "All users"}}
	})
}

func TestDataSources(t *testing.T) {
	f := newFakeJira(t)
	withSchemes(f)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{{
			Config: providerBlock(f) + `
data "jira_issue_types_schema" "default" {
  name = "Default Issue Type Scheme"
}

data "jira_priority_types_schema" "default" {
  name = "Default Priority Scheme"
}

data "jira_permission_schema" "default" {
  name = "Default Permission Scheme"
}

data "jira_workflow_schema" "by_id" {
  id = 11601
}

data "jira_workflow_schema" "by_name" {
  name = "Default Workflow Scheme"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.jira_issue_types_schema.default", "id", "11947"),
				resource.TestCheckResourceAttr("data.jira_priority_types_schema.default", "id", "11907"),
				resource.TestCheckResourceAttr("data.jira_priority_types_schema.default", "options_ids.#", "3"),
				resource.TestCheckResourceAttr("data.jira_permission_schema.default", "id", "10600"),
				resource.TestCheckResourceAttr("data.jira_workflow_schema.by_id", "name", "Default Workflow Scheme"),
				resource.TestCheckResourceAttr("data.jira_workflow_schema.by_name", "id", "11601"),
				// Both forms of the workflow lookup end in the same state.
				resource.TestCheckResourceAttrPair(
					"data.jira_workflow_schema.by_id", "self",
					"data.jira_workflow_schema.by_name", "self"),
			),
		}},
	})
}

func TestDataSources_NotFound(t *testing.T) {
	f := newFakeJira(t)
	withSchemes(f)

	for name, config := range map[string]string{
		"issue type scheme": `data "jira_issue_types_schema" "x" { name = "Missing" }`,
		"priority scheme":   `data "jira_priority_types_schema" "x" { name = "Missing" }`,
		"permission scheme": `data "jira_permission_schema" "x" { name = "Missing" }`,
		"workflow by name":  `data "jira_workflow_schema" "x" { name = "Missing" }`,
		"workflow by id":    `data "jira_workflow_schema" "x" { id = 99999 }`,
	} {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: providerFactories(),
				Steps: []resource.TestStep{{
					Config:      providerBlock(f) + config,
					ExpectError: regexp.MustCompile(`not found|does not exist`),
				}},
			})
		})
	}
}

// Credentials left out of the configuration come from JIRA_USERNAME and JIRA_TOKEN.
func TestProvider_CredentialsFromEnvironment(t *testing.T) {
	f := newFakeJira(t)
	withSchemes(f)
	t.Setenv("JIRA_USERNAME", "env-user")
	t.Setenv("JIRA_TOKEN", "env-token")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{{
			Config: `
provider "jira" {
  base_url = "` + f.URL() + `"
}

data "jira_permission_schema" "default" {
  name = "Default Permission Scheme"
}
`,
			Check: func(*terraform.State) error {
				if got := f.user(); got != "env-user" {
					return fmt.Errorf("Jira saw user %q, want the one from JIRA_USERNAME", got)
				}
				return nil
			},
		}},
	})
}
