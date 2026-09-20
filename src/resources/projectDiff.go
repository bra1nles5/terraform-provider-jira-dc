package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// isSet reports whether the configuration actually carries a value for an
// Optional+Computed attribute.
func isSet(value types.Int32) bool {
	return !value.IsNull() && !value.IsUnknown()
}

// resolveSchemaID replaces an unknown attribute with what Jira reports. Archived
// projects refuse to report, and there null is the only truthful value.
func resolveSchemaID(ctx context.Context, planned types.Int32, read func() (int32, error)) types.Int32 {
	if isSet(planned) {
		return planned
	}
	id, err := read()
	if err != nil {
		tflog.Warn(ctx, "schema id not readable after write, storing null", map[string]any{"error": err.Error()})
		return types.Int32Null()
	}
	return types.Int32Value(id)
}

type projectDiff struct {
	plan  ProjectResourceModel
	state ProjectResourceModel
}

func newProjectDiff(plan, state ProjectResourceModel) *projectDiff {
	return &projectDiff{plan: plan, state: state}
}

func (d *projectDiff) NameChanged() bool {
	return !d.plan.Name.Equal(d.state.Name)
}

func (d *projectDiff) TypeChanged() bool {
	return !d.plan.Type.Equal(d.state.Type)
}

func (d *projectDiff) LeadChanged() bool {
	return !d.plan.Lead.Equal(d.state.Lead)
}

func (d *projectDiff) DescriptionChanged() bool {
	return !d.plan.Description.Equal(d.state.Description)
}

func (d *projectDiff) ArchivedChanged() bool {
	return !d.plan.Archived.Equal(d.state.Archived)
}

func (d *projectDiff) IssueTypeChanged() bool {
	return !d.plan.IssueTypeSchemaId.Equal(d.state.IssueTypeSchemaId)
}

func (d *projectDiff) PriorityChanged() bool {
	return !d.plan.PrioritySchemaId.Equal(d.state.PrioritySchemaId)
}

func (d *projectDiff) PermissionChanged() bool {
	return !d.plan.PermissionSchemaId.Equal(d.state.PermissionSchemaId)
}

func (d *projectDiff) WorkflowChanged() bool {
	return !d.plan.WorkflowSchemaId.Equal(d.state.WorkflowSchemaId)
}

func (d *projectDiff) BaseChanged() bool {
	return d.NameChanged() || d.DescriptionChanged() || d.LeadChanged()
}

// retryAfterCreate repeats an operation that may fail right after the project is
// created: Jira finishes creating it asynchronously, and for the first seconds
// permission checks against it do not pass. Five attempts, 2 seconds apart.
func retryAfterCreate(ctx context.Context, action string, do func() error) error {
	const attempts = 5

	var err error
	for i := 0; i < attempts; i++ {
		if err = do(); err == nil {
			return nil
		}
		tflog.Warn(ctx, "retrying after project creation", map[string]any{
			"action": action, "attempt": i + 1, "error": err.Error(),
		})
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
