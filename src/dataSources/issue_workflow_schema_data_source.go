package datasources

import (
	"context"
	"fmt"
	jira "github.com/bra1nles5/terraform-provider-jira-dc/src/jiraClient"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IssueWorkflowSchemaDataSource struct {
	client *jira.JiraClient
}

type IssueWorkflowSchemaModel struct {
	ID          types.Int32  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Self        types.String `tfsdk:"self"`
}

func NewWorkflowSchemaDataSource() datasource.DataSource {
	return &IssueWorkflowSchemaDataSource{}
}

func (d *IssueWorkflowSchemaDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "jira_workflow_schema"
}

func (d *IssueWorkflowSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Jira workflow scheme by name or by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Uniq id. Set either this or name.",
				Validators: []validator.Int32{
					int32validator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Schemas name. Set either this or id.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Issue Priority Schema Description.",
			},
			"self": schema.StringAttribute{
				Computed:    true,
				Description: "Issue Priority Schema link.",
			},
		},
	}
}

func (d *IssueWorkflowSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, resp)
}

func (d *IssueWorkflowSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state IssueWorkflowSchemaModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueInt32()

	// Looking up by name resolves to an ID first; the canonical fields then come from
	// the same REST call the ID form uses, so both forms produce identical state.
	if !state.Name.IsNull() && !state.Name.IsUnknown() {
		cache, err := d.client.Workflow.GetAll(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Failed to load workflow schemas", err.Error())
			return
		}

		scheme, ok := cache[state.Name.ValueString()]
		if !ok {
			resp.Diagnostics.AddError(
				"Workflow schema not found",
				fmt.Sprintf("No workflow schema with name %q", state.Name.ValueString()),
			)
			return
		}
		id = scheme.ID
	}

	its, err := d.client.Workflow.Get(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get workflow schema", err.Error())
		return
	}

	state.ID = types.Int32Value(id)
	state.Name = types.StringValue(its.Name)
	state.Self = types.StringValue(its.Self)
	state.Description = types.StringValue(its.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
