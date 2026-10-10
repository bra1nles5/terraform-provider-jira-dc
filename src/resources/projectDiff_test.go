package resources

import (
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKeepLeadCase(t *testing.T) {
	tests := []struct {
		name     string
		known    types.String
		fromJira string
		want     types.String
	}{
		{"same spelling", types.StringValue("jsmith"), "jsmith", types.StringValue("jsmith")},
		{"other case keeps configuration", types.StringValue("jsmith"), "JSmith", types.StringValue("jsmith")},
		{"another user comes from Jira", types.StringValue("jsmith"), "adoe", types.StringValue("adoe")},
		{"null after import comes from Jira", types.StringNull(), "JSmith", types.StringValue("JSmith")},
		{"unknown comes from Jira", types.StringUnknown(), "JSmith", types.StringValue("JSmith")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keepLeadCase(tt.known, tt.fromJira); !got.Equal(tt.want) {
				t.Errorf("keepLeadCase(%s, %q) = %s, want %s", tt.known, tt.fromJira, got, tt.want)
			}
		})
	}
}

// archivedModel is a project as an import of an archived one leaves it in state:
// Jira hides its workflow, priority and permission schemes, so those are null.
func archivedModel() ProjectResourceModel {
	return ProjectResourceModel{
		Key:                types.StringValue("EX"),
		Name:               types.StringValue("Example"),
		Description:        types.StringNull(),
		Type:               types.StringValue("software"),
		Lead:               types.StringValue("jsmith"),
		Archived:           types.BoolValue(true),
		IssueTypeSchemaId:  types.StringValue("15408"),
		WorkflowSchemaId:   types.Int32Null(),
		PrioritySchemaId:   types.Int32Null(),
		PermissionSchemaId: types.Int32Null(),
	}
}

func TestProjectDiff_WhileArchived(t *testing.T) {
	tests := []struct {
		name         string
		state, plan  func(*ProjectResourceModel)
		staysArchive bool
		adopted      []string
		refused      []string
	}{
		{
			name:         "nothing changes",
			staysArchive: true,
		},
		{
			name: "hidden schemes set after import are adopted",
			plan: func(m *ProjectResourceModel) {
				m.WorkflowSchemaId = types.Int32Value(11601)
				m.PrioritySchemaId = types.Int32Value(11907)
				m.PermissionSchemaId = types.Int32Value(10600)
			},
			staysArchive: true,
			adopted:      []string{"workflow_schema_id", "priority_schema_id", "permission_schema_id"},
		},
		{
			name:         "issue type scheme may change: Jira accepts it",
			plan:         func(m *ProjectResourceModel) { m.IssueTypeSchemaId = types.StringValue("11947") },
			staysArchive: true,
		},
		{
			name:         "a known hidden scheme cannot change",
			state:        func(m *ProjectResourceModel) { m.PrioritySchemaId = types.Int32Value(10100) },
			plan:         func(m *ProjectResourceModel) { m.PrioritySchemaId = types.Int32Value(11907) },
			staysArchive: true,
			refused:      []string{"priority_schema_id"},
		},
		{
			name: "base fields cannot change",
			plan: func(m *ProjectResourceModel) {
				m.Name = types.StringValue("Renamed")
				m.Description = types.StringValue("New")
				m.Lead = types.StringValue("adoe")
				m.Type = types.StringValue("business")
			},
			staysArchive: true,
			refused:      []string{"name", "description", "lead", "type"},
		},
		{
			name: "adoption and refusal are told apart",
			plan: func(m *ProjectResourceModel) {
				m.Name = types.StringValue("Renamed")
				m.WorkflowSchemaId = types.Int32Value(11601)
			},
			staysArchive: true,
			adopted:      []string{"workflow_schema_id"},
			refused:      []string{"name"},
		},
		{
			name: "a project being restored is not archived for the update",
			plan: func(m *ProjectResourceModel) {
				m.Archived = types.BoolValue(false)
				m.Name = types.StringValue("Renamed")
			},
		},
		{
			name: "a project being archived is not archived yet",
			state: func(m *ProjectResourceModel) {
				m.Archived = types.BoolValue(false)
			},
			plan: func(m *ProjectResourceModel) { m.Name = types.StringValue("Renamed") },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, plan := archivedModel(), archivedModel()
			if tt.state != nil {
				tt.state(&state)
				plan = state // the plan starts from state and changes only what the case sets
			}
			if tt.plan != nil {
				tt.plan(&plan)
			}
			d := newProjectDiff(plan, state)

			if got := d.StaysArchived(); got != tt.staysArchive {
				t.Errorf("StaysArchived() = %v, want %v", got, tt.staysArchive)
			}
			if !tt.staysArchive {
				return
			}
			if got := d.AdoptedWhileArchived(); !slices.Equal(got, tt.adopted) {
				t.Errorf("AdoptedWhileArchived() = %v, want %v", got, tt.adopted)
			}
			if got := d.RefusedWhileArchived(); !slices.Equal(got, tt.refused) {
				t.Errorf("RefusedWhileArchived() = %v, want %v", got, tt.refused)
			}
		})
	}
}
