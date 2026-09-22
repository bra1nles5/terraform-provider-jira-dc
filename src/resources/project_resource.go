package resources

import (
	"context"
	"fmt"
	jira "github.com/bra1nles5/terraform-provider-jira-dc/src/jiraClient"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ProjectResource struct {
	projectService *jira.ProjectService
}

type ProjectResourceModel struct {
	Key                types.String `tfsdk:"key"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Type               types.String `tfsdk:"type"`
	Lead               types.String `tfsdk:"lead"`
	Archived           types.Bool   `tfsdk:"archived"`
	IssueTypeSchemaId  types.String `tfsdk:"issue_type_schema_id"`
	WorkflowSchemaId   types.Int32  `tfsdk:"workflow_schema_id"`
	PrioritySchemaId   types.Int32  `tfsdk:"priority_schema_id"`
	PermissionSchemaId types.Int32  `tfsdk:"permission_schema_id"`
}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

func (r *ProjectResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "jira_project"
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*jira.JiraClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			fmt.Sprintf("Expected *JiraClient, got: %T", req.ProviderData),
		)
		return
	}
	r.projectService = &jira.ProjectService{
		Client: client,
	}
}

func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Jira Data Center project together with the schemes it is built from: " +
			"issue types, workflow, priorities and permissions.",
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Required: true,
				Description: "Project key, unique across the Jira instance, for example `EX`. " +
					"Jira does not allow changing it, so a new value replaces the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Project display name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Project description. Left unset it stays empty in Jira.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Project type: `software`, `business` or `service_desk`.",
				Validators: []validator.String{
					stringvalidator.OneOf("software", "business", "service_desk"),
				},
			},
			"lead": schema.StringAttribute{
				Required:    true,
				Description: "Username of the project lead.",
			},
			"archived": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Whether the project is archived. Jira ignores this field on create and update, " +
					"so it is applied with a separate archive or restore call afterwards.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"issue_type_schema_id": schema.StringAttribute{
				Required: true,
				Description: "ID of the issue type scheme to associate with the project. " +
					"It is a string because that is how Jira reports it for this scheme.",
			},
			"workflow_schema_id": schema.Int32Attribute{
				Required: true,
				Description: "ID of the workflow scheme to associate with the project. Assigning it goes " +
					"through the `assignWorkflowScheme` ScriptRunner endpoint; on a project that already " +
					"has issues Jira migrates them, which may take a while.",
			},
			// Optional rather than Required: an archived project does not expose these
			// two, so a configuration describing one has nothing to put here.
			"priority_schema_id": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Description: "ID of the priority scheme. Optional because an archived project refuses to " +
					"report it: in that case the value is null rather than a guess.",
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"permission_schema_id": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Description: "ID of the permission scheme. Sent already when the project is created: " +
					"without it not even an administrator can see the new project. An archived project " +
					"does not report it, and the value is then null.",
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prj := jira.Project{
		Key:         plan.Key.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Lead:        plan.Lead.ValueString(),
		Type:        plan.Type.ValueString(),
		Archived:    plan.Archived.ValueBool(),
	}
	// The permission scheme is needed at creation time: without it not even an
	// administrator can see the project, and assigning the other schemes fails with 400.
	if isSet(plan.PermissionSchemaId) {
		prj.PermissionScheme = plan.PermissionSchemaId.ValueInt32()
	}

	_, err := r.projectService.Create(ctx, prj)
	if err != nil {
		resp.Diagnostics.AddError("Error creating project", err.Error())
		return
	}

	// Right after POST /project the permission check does not see the project yet:
	// assigning a scheme fails with 400 "You must have the view project permission".
	// The same call succeeds a few seconds later, so the first assignments are retried.
	err = retryAfterCreate(ctx, "assign issue type schema", func() error {
		return r.projectService.AssignIssueTypeSchema(ctx, prj.Key, plan.IssueTypeSchemaId.ValueString())
	})
	if err != nil {
		resp.Diagnostics.AddError("Error assigning issue type schema", err.Error())
		return
	}

	if isSet(plan.PrioritySchemaId) {
		err = r.projectService.AssignPrioritySchema(ctx, prj.Key, fmt.Sprintf("%d", plan.PrioritySchemaId.ValueInt32()))
		if err != nil {
			resp.Diagnostics.AddError("Error assigning priority schema", err.Error())
			return
		}
	}

	if isSet(plan.PermissionSchemaId) {
		err = r.projectService.AssignPermissionSchema(ctx, &prj, plan.PermissionSchemaId.ValueInt32())
		if err != nil {
			resp.Diagnostics.AddError("Error assigning permission schema", err.Error())
			return
		}
	}

	err = r.projectService.AssignWorkflow(ctx, prj.Key, plan.WorkflowSchemaId.ValueInt32(), true)
	if err != nil {
		resp.Diagnostics.AddError("Error assigning workflow", err.Error())
		return
	}

	// Schemes are checked before archiving: an archived project may hide them.
	if err := r.reconcileSchemas(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Schemas did not survive project creation", err.Error())
		return
	}

	if plan.Archived.ValueBool() {
		if err := r.projectService.Archive(ctx, prj.Key); err != nil {
			resp.Diagnostics.AddError("Error archiving project after create", err.Error())
			return
		}
		tflog.Debug(ctx, "project archived after create", map[string]any{"key": prj.Key})
	}

	canonical, err := r.projectService.Get(ctx, prj.Key)
	if err != nil {
		resp.Diagnostics.AddError("Error reading project after create", err.Error())
		return
	}
	plan.Key = types.StringValue(canonical.Key)
	plan.Name = types.StringValue(canonical.Name)
	plan.Lead = keepLeadCase(plan.Lead, canonical.Lead)
	plan.Type = types.StringValue(canonical.Type)
	plan.Archived = types.BoolValue(canonical.Archived)
	if canonical.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(canonical.Description)
	}

	// Both are Computed, so state must not keep an unknown value. Read back what Jira
	// ended up with; an archived project hides them, and then null is the honest answer.
	plan.PrioritySchemaId = resolveSchemaID(ctx, plan.PrioritySchemaId, func() (int32, error) {
		return r.projectService.GetPrioritySchema(ctx, prj.Key)
	})
	plan.PermissionSchemaId = resolveSchemaID(ctx, plan.PermissionSchemaId, func() (int32, error) {
		return r.projectService.GetPermissionSchema(ctx, prj.Key)
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := state.Key.ValueString()

	prj, err := r.projectService.Get(ctx, key)
	if err != nil || prj == nil {
		if jira.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if err != nil {
			resp.Diagnostics.AddError("Error reading project", err.Error())
		} else {
			resp.Diagnostics.AddError("Error reading project", "Jira returned no project")
		}
		return
	}

	state.Key = types.StringValue(prj.Key)
	state.Name = types.StringValue(prj.Name)
	state.Lead = keepLeadCase(state.Lead, prj.Lead)
	state.Type = types.StringValue(prj.Type)
	state.Archived = types.BoolValue(prj.Archived)
	if prj.Description == "" {
		state.Description = types.StringNull()
	} else {
		state.Description = types.StringValue(prj.Description)
	}

	issueTypeID, err := r.projectService.GetIssueTypeSchema(ctx, key)
	if err != nil {
		resp.Diagnostics.AddError("Error reading issue type schema", err.Error())
		return
	}
	state.IssueTypeSchemaId = types.StringValue(issueTypeID)

	workflowID, err := r.projectService.GetWorkflowSchema(ctx, key)
	switch {
	case err == nil:
		state.WorkflowSchemaId = types.Int32Value(workflowID)
	case jira.IsForbidden(err):
		tflog.Warn(ctx, "workflow schema not readable, keeping state", map[string]any{"key": key})
	default:
		resp.Diagnostics.AddError("Error reading workflow schema", err.Error())
		return
	}

	// Jira answers 403 for the priority and permission schemes of an archived project
	// ("You cannot view this project"). That is not a failure: the schemes stay as
	// they are, they just cannot be observed. Keep whatever state already holds.
	priorityID, err := r.projectService.GetPrioritySchema(ctx, key)
	switch {
	case err == nil:
		state.PrioritySchemaId = types.Int32Value(priorityID)
	case jira.IsForbidden(err):
		tflog.Warn(ctx, "priority schema not readable, keeping state", map[string]any{"key": key})
	default:
		resp.Diagnostics.AddError("Error reading priority schema", err.Error())
		return
	}

	permissionID, err := r.projectService.GetPermissionSchema(ctx, key)
	switch {
	case err == nil:
		state.PermissionSchemaId = types.Int32Value(permissionID)
	case jira.IsForbidden(err):
		tflog.Warn(ctx, "permission schema not readable, keeping state", map[string]any{"key": key})
	default:
		resp.Diagnostics.AddError("Error reading permission schema", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	diff := newProjectDiff(plan, state)

	prj := jira.Project{
		Key:         plan.Key.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Lead:        plan.Lead.ValueString(),
		Type:        plan.Type.ValueString(),
		Archived:    plan.Archived.ValueBool(),
	}

	tflog.Debug(ctx, "starting project update", map[string]any{"key": prj.Key})

	// An archived project refuses changes, so restoring comes first and archiving last.
	if diff.ArchivedChanged() && !plan.Archived.ValueBool() {
		if err := r.projectService.Restore(ctx, prj.Key); err != nil {
			resp.Diagnostics.AddError("Error restoring project", err.Error())
			return
		}
		tflog.Debug(ctx, "project restored", map[string]any{"key": prj.Key})
	}

	if diff.BaseChanged() {
		if err := r.projectService.Update(ctx, &prj); err != nil {
			resp.Diagnostics.AddError("Error updating project", err.Error())
			return
		}
		tflog.Debug(ctx, "project base fields updated", map[string]any{"key": prj.Key})
	}

	if diff.TypeChanged() {
		if err := r.projectService.UpdateProjectType(ctx, prj.Key, plan.Type.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error updating project type", err.Error())
			return
		}
		tflog.Debug(ctx, "project type updated", map[string]any{"key": prj.Key})
	}

	if diff.IssueTypeChanged() {
		if err := r.projectService.AssignIssueTypeSchema(ctx, prj.Key, plan.IssueTypeSchemaId.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error updating issue type schema", err.Error())
			return
		}
		tflog.Debug(ctx, "project issue type schema updated", map[string]any{"key": prj.Key})
	}

	if diff.PriorityChanged() && isSet(plan.PrioritySchemaId) {
		if err := r.projectService.AssignPrioritySchema(ctx, prj.Key, fmt.Sprintf("%d", plan.PrioritySchemaId.ValueInt32())); err != nil {
			resp.Diagnostics.AddError("Error updating priority schema", err.Error())
			return
		}
		tflog.Debug(ctx, "project priority schema updated", map[string]any{"key": prj.Key})
	}

	if diff.PermissionChanged() && isSet(plan.PermissionSchemaId) {
		if err := r.projectService.AssignPermissionSchema(ctx, &prj, plan.PermissionSchemaId.ValueInt32()); err != nil {
			resp.Diagnostics.AddError("Error updating permission schema", err.Error())
			return
		}
		tflog.Debug(ctx, "project permission schema updated", map[string]any{"key": prj.Key})
	}

	if diff.WorkflowChanged() {
		if err := r.projectService.AssignWorkflow(ctx, prj.Key, plan.WorkflowSchemaId.ValueInt32(), true); err != nil {
			resp.Diagnostics.AddError("Error updating workflow", err.Error())
			return
		}
		tflog.Debug(ctx, "project workflow schema updated", map[string]any{"key": prj.Key})
	}

	if diff.ArchivedChanged() && plan.Archived.ValueBool() {
		if err := r.projectService.Archive(ctx, prj.Key); err != nil {
			resp.Diagnostics.AddError("Error archiving project", err.Error())
			return
		}
		tflog.Debug(ctx, "project archived", map[string]any{"key": prj.Key})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("key"), req, resp)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prj := jira.Project{
		Key:         plan.Key.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Type:        plan.Type.ValueString(),
		Archived:    plan.Archived.ValueBool(),
	}

	err := r.projectService.Remove(ctx, &prj)
	if jira.IsNotFound(err) {
		tflog.Warn(ctx, "project already gone, nothing to delete", map[string]any{"key": prj.Key})
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error removing project", err.Error())
		return
	}
}

// reconcileSchemas verifies that Jira kept what was assigned.
//
// Jira finishes applying the project template after it answers POST /project and
// may replace the schemes just assigned with freshly created ones of its own. The
// assignment still returns 200, so the only way to catch the swap is to read the
// actual state back and assign again on mismatch.
func (r *ProjectResource) reconcileSchemas(ctx context.Context, plan ProjectResourceModel) error {
	key := plan.Key.ValueString()

	issueType, err := r.projectService.GetIssueTypeSchema(ctx, key)
	if err != nil {
		return fmt.Errorf("reading issue type schema back: %w", err)
	}
	if issueType != plan.IssueTypeSchemaId.ValueString() {
		tflog.Warn(ctx, "issue type schema was overwritten by the project template, reassigning",
			map[string]any{"key": key, "got": issueType, "want": plan.IssueTypeSchemaId.ValueString()})
		if err := r.projectService.AssignIssueTypeSchema(ctx, key, plan.IssueTypeSchemaId.ValueString()); err != nil {
			return err
		}
		if issueType, err = r.projectService.GetIssueTypeSchema(ctx, key); err != nil {
			return fmt.Errorf("reading issue type schema back: %w", err)
		}
		if issueType != plan.IssueTypeSchemaId.ValueString() {
			return fmt.Errorf("issue type schema stayed %s after reassigning to %s",
				issueType, plan.IssueTypeSchemaId.ValueString())
		}
	}

	workflow, err := r.projectService.GetWorkflowSchema(ctx, key)
	if err != nil {
		return fmt.Errorf("reading workflow schema back: %w", err)
	}
	if workflow != plan.WorkflowSchemaId.ValueInt32() {
		tflog.Warn(ctx, "workflow schema was overwritten by the project template, reassigning",
			map[string]any{"key": key, "got": workflow, "want": plan.WorkflowSchemaId.ValueInt32()})
		if err := r.projectService.AssignWorkflow(ctx, key, plan.WorkflowSchemaId.ValueInt32(), true); err != nil {
			return err
		}
		if workflow, err = r.projectService.GetWorkflowSchema(ctx, key); err != nil {
			return fmt.Errorf("reading workflow schema back: %w", err)
		}
		if workflow != plan.WorkflowSchemaId.ValueInt32() {
			return fmt.Errorf("workflow schema stayed %d after reassigning to %d",
				workflow, plan.WorkflowSchemaId.ValueInt32())
		}
	}

	return nil
}
