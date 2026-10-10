package src

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These tests run real Terraform against the provider and a fake Jira (see
// jirafake_test.go). After every step the framework also checks that a second plan
// is empty, so each scenario proves the configuration converges.

func providerFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"jira": providerserver.NewProtocol6WithError(NewProvider("test")),
	}
}

func providerBlock(f *fakeJira) string {
	return fmt.Sprintf(`
provider "jira" {
  base_url = %q
  username = "admin"
  token    = "test"
}
`, f.URL())
}

// checkJira checks the project as the fake Jira holds it, not as state claims.
func checkJira(f *fakeJira, key string, check func(p fakeProject) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		p := f.project(key)
		if p == nil {
			return fmt.Errorf("project %s is not in Jira", key)
		}
		return check(*p)
	}
}

func checkGone(f *fakeJira, key string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if f.project(key) != nil {
			return fmt.Errorf("project %s is still in Jira", key)
		}
		return nil
	}
}

func TestProjectResource_Lifecycle(t *testing.T) {
	f := newFakeJira(t)
	config := func(name string, priority, permission int, extra string) string {
		return providerBlock(f) + fmt.Sprintf(`
resource "jira_project" "test" {
  key                  = "EX"
  name                 = %q
  type                 = "software"
  lead                 = "jsmith"
  issue_type_schema_id = "11947"
  priority_schema_id   = %d
  permission_schema_id = %d
  %s
}
`, name, priority, permission, extra)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{
			{
				// workflow_schema_id is not set, so the project keeps the template's scheme.
				Config: config("Example", 11907, 10600, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jira_project.test", "workflow_schema_id", fmt.Sprint(defaultWorkflowScheme)),
					resource.TestCheckResourceAttr("jira_project.test", "archived", "false"),
					checkJira(f, "EX", func(p fakeProject) error {
						if p.IssueTypeScheme != "11947" || p.WorkflowScheme != defaultWorkflowScheme ||
							p.PriorityScheme != 11907 || p.PermissionScheme != 10600 {
							return fmt.Errorf("unexpected schemes in Jira: %+v", p)
						}
						return nil
					}),
				),
			},
			{
				Config: config("Renamed", 10200, 10601, `description = "Managed by Terraform"`),
				Check: checkJira(f, "EX", func(p fakeProject) error {
					if p.Name != "Renamed" || p.Description != "Managed by Terraform" ||
						p.PriorityScheme != 10200 || p.PermissionScheme != 10601 {
						return fmt.Errorf("update did not reach Jira: %+v", p)
					}
					return nil
				}),
			},
			{
				// Removing the description empties it in Jira and leaves null in state.
				Config: config("Renamed", 10200, 10601, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("jira_project.test", "description"),
					checkJira(f, "EX", func(p fakeProject) error {
						if p.Description != "" {
							return fmt.Errorf("description was not cleared in Jira: %+v", p)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:                         "jira_project.test",
				ImportState:                          true,
				ImportStateId:                        "EX",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
			},
		},
	})
}

// zooProject is an archived project as found in Jira before Terraform manages it.
func zooProject() fakeProject {
	return fakeProject{
		Key: "EX", Name: "Zoo", Type: "software", Lead: "admin", Archived: true,
		IssueTypeScheme: "15408", WorkflowScheme: 14608, PriorityScheme: 10100, PermissionScheme: 10600,
	}
}

// TestProjectResource_ArchivedProject follows the fleet import that used to fail:
// one configuration assigns the same schemes to every project, archived ones included.
func TestProjectResource_ArchivedProject(t *testing.T) {
	f := newFakeJira(t)
	f.put(zooProject())

	config := func(name string, archived bool) string {
		return providerBlock(f) + fmt.Sprintf(`
import {
  to = jira_project.test
  id = "EX"
}

resource "jira_project" "test" {
  key                  = "EX"
  name                 = %q
  type                 = "software"
  lead                 = "admin"
  archived             = %t
  issue_type_schema_id = "11947"
  workflow_schema_id   = 11601
  priority_schema_id   = 11907
  permission_schema_id = 10600
}
`, name, archived)
	}

	var mark int
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{
			{
				// Jira takes the issue type scheme. The hidden schemes it neither shows nor
				// changes are recorded as configured, and the next plan is empty.
				Config: config("Zoo", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jira_project.test", "workflow_schema_id", "11601"),
					resource.TestCheckResourceAttr("jira_project.test", "priority_schema_id", "11907"),
					resource.TestCheckResourceAttr("jira_project.test", "permission_schema_id", "10600"),
					checkJira(f, "EX", func(p fakeProject) error {
						if !p.Archived || p.IssueTypeScheme != "11947" || p.WorkflowScheme != 14608 || p.PriorityScheme != 10100 {
							return fmt.Errorf("expected only the issue type scheme to change in Jira: %+v", p)
						}
						return nil
					}),
				),
			},
			{
				// Renaming an archived project fails before anything reaches Jira.
				PreConfig:   func() { mark = f.writeCount() },
				Config:      config("Renamed", true),
				ExpectError: regexp.MustCompile(`Project is archived`),
			},
			{
				// Restoring reads the real schemes back and assigns the configured ones.
				PreConfig: func() {
					if writes := f.writesSince(mark); len(writes) > 0 {
						t.Errorf("the refused update still wrote to Jira: %v", writes)
					}
				},
				Config: config("Zoo", false),
				Check: checkJira(f, "EX", func(p fakeProject) error {
					if p.Archived || p.Name != "Zoo" || p.IssueTypeScheme != "11947" ||
						p.WorkflowScheme != 11601 || p.PriorityScheme != 11907 || p.PermissionScheme != 10600 {
						return fmt.Errorf("restored project does not match the configuration: %+v", p)
					}
					return nil
				}),
			},
		},
	})
}

// TestProjectResource_ArchivedProjectGeneratedConfig uses what
// `terraform plan -generate-config-out` writes for an archived project: the hidden
// schemes come out null. Such a configuration is valid and imports without changes.
func TestProjectResource_ArchivedProjectGeneratedConfig(t *testing.T) {
	f := newFakeJira(t)
	f.put(zooProject())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(f) + `
import {
  to = jira_project.test
  id = "EX"
}

resource "jira_project" "test" {
  archived             = true
  description          = null
  issue_type_schema_id = "15408"
  key                  = "EX"
  lead                 = "admin"
  name                 = "Zoo"
  type                 = "software"
  workflow_schema_id   = null
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("jira_project.test", "workflow_schema_id"),
					func(*terraform.State) error {
						if writes := f.writesSince(0); len(writes) > 0 {
							return fmt.Errorf("the import wrote to Jira: %v", writes)
						}
						return nil
					},
				),
			},
		},
	})
}

// projectConfig is a project with every scheme set, plus extra attributes.
func projectConfig(f *fakeJira, extra string) string {
	return providerBlock(f) + fmt.Sprintf(`
resource "jira_project" "test" {
  key                  = "EX"
  name                 = "Example"
  type                 = "software"
  issue_type_schema_id = "11947"
  workflow_schema_id   = 11601
  priority_schema_id   = 11907
  permission_schema_id = 10600
%s
}
`, extra)
}

// Jira finishes applying the project template after answering POST /project and may
// replace the schemes just assigned. The provider reads them back and assigns again.
func TestProjectResource_TemplateOverwritesSchemes(t *testing.T) {
	f := newFakeJira(t)
	f.set(func(f *fakeJira) { f.templateOverwrites = true })

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{{
			Config: projectConfig(f, `lead = "jsmith"`),
			Check: checkJira(f, "EX", func(p fakeProject) error {
				if p.IssueTypeScheme != "11947" || p.WorkflowScheme != 11601 {
					return fmt.Errorf("template schemes were left in place: %+v", p)
				}
				return nil
			}),
		}},
	})
}

// Right after POST /project Jira refuses assignments for a few seconds with 400
// "You must have the view project permission"; the provider retries.
func TestProjectResource_RetriesRightAfterCreate(t *testing.T) {
	f := newFakeJira(t)
	f.set(func(f *fakeJira) { f.createLag = 1 })

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{{
			Config: projectConfig(f, `lead = "jsmith"`),
			Check: checkJira(f, "EX", func(p fakeProject) error {
				if p.IssueTypeScheme != "11947" {
					return fmt.Errorf("issue type scheme was not assigned: %+v", p)
				}
				return nil
			}),
		}},
	})
}

// Jira spells user names its own way. A lead that differs only in case is kept as
// configured: it used to fail apply with "inconsistent result" (v0.1.2).
func TestProjectResource_LeadInAnotherCase(t *testing.T) {
	f := newFakeJira(t)
	f.set(func(f *fakeJira) { f.leadSpelling = map[string]string{"jsmith": "JSmith"} })

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{{
			Config: projectConfig(f, `lead = "jsmith"`),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("jira_project.test", "lead", "jsmith"),
				checkJira(f, "EX", func(p fakeProject) error {
					if p.Lead != "JSmith" {
						return fmt.Errorf("the fake did not respell the lead: %+v", p)
					}
					return nil
				}),
			),
		}},
	})
}

// Schemes are assigned and checked before a project created as archived is archived.
func TestProjectResource_CreateArchived(t *testing.T) {
	f := newFakeJira(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{{
			Config: projectConfig(f, "lead = \"jsmith\"\n  archived = true"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("jira_project.test", "archived", "true"),
				resource.TestCheckResourceAttr("jira_project.test", "priority_schema_id", "11907"),
				checkJira(f, "EX", func(p fakeProject) error {
					if !p.Archived || p.WorkflowScheme != 11601 || p.PriorityScheme != 11907 || p.PermissionScheme != 10600 {
						return fmt.Errorf("schemes were not in place before archiving: %+v", p)
					}
					return nil
				}),
			),
		}},
	})
}

func TestProjectResource_ChangeTypeLeadAndArchive(t *testing.T) {
	f := newFakeJira(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{
			{Config: projectConfig(f, `lead = "jsmith"`)},
			{
				Config: strings.Replace(projectConfig(f, `lead = "adoe"`), `"software"`, `"business"`, 1),
				Check: checkJira(f, "EX", func(p fakeProject) error {
					if p.Type != "business" || p.Lead != "adoe" {
						return fmt.Errorf("type or lead did not change in Jira: %+v", p)
					}
					return nil
				}),
			},
			{
				Config: strings.Replace(projectConfig(f, "lead = \"adoe\"\n  archived = true"), `"software"`, `"business"`, 1),
				Check: checkJira(f, "EX", func(p fakeProject) error {
					if !p.Archived {
						return fmt.Errorf("project was not archived: %+v", p)
					}
					return nil
				}),
			},
		},
	})
}

// A project deleted by hand disappears from state, and the next apply creates it again.
func TestProjectResource_DeletedOutsideTerraform(t *testing.T) {
	f := newFakeJira(t)
	config := projectConfig(f, `lead = "jsmith"`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() { f.set(func(f *fakeJira) { delete(f.projects, "EX") }) },
				Config:    config,
				Check: checkJira(f, "EX", func(fakeProject) error {
					return nil // checkJira fails by itself when the project is missing
				}),
			},
		},
	})
}

// When Jira refuses to delete, destroy fails and the project stays in state; it
// used to vanish from state while staying in Jira (v0.1.1).
func TestProjectResource_DeleteRefused(t *testing.T) {
	f := newFakeJira(t)
	config := projectConfig(f, `lead = "jsmith"`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig:   func() { f.set(func(f *fakeJira) { f.refuseDelete = true }) },
				Config:      config,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Error removing project`),
			},
			{
				// Still in state and in Jira: the plan for the same configuration is empty.
				PreConfig: func() { f.set(func(f *fakeJira) { f.refuseDelete = false }) },
				Config:    config,
				Check: checkJira(f, "EX", func(fakeProject) error {
					return nil
				}),
			},
		},
	})
}

// With only the required attributes, the hidden schemes come from the project
// template and the plan still converges.
func TestProjectResource_MinimalConfig(t *testing.T) {
	f := newFakeJira(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		CheckDestroy:             checkGone(f, "EX"),
		Steps: []resource.TestStep{{
			Config: providerBlock(f) + `
resource "jira_project" "test" {
  key                  = "EX"
  name                 = "Example"
  type                 = "software"
  lead                 = "jsmith"
  issue_type_schema_id = "11947"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("jira_project.test", "workflow_schema_id", fmt.Sprint(defaultWorkflowScheme)),
				resource.TestCheckResourceAttr("jira_project.test", "priority_schema_id", fmt.Sprint(defaultPriorityScheme)),
				resource.TestCheckResourceAttr("jira_project.test", "permission_schema_id", fmt.Sprint(defaultPermissionScheme)),
				resource.TestCheckResourceAttr("jira_project.test", "archived", "false"),
			),
		}},
	})
}
