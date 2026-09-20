package datasources

import (
	"context"
	"fmt"
	jira "github.com/bra1nles5/terraform-provider-jira-dc/src/jiraClient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IssueTypesSchemaDataSource struct {
	client *jira.JiraClient
}

type IssueTypesSchemaModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Self        types.String `tfsdk:"self"`
}

func NewIssueTypesSchemaDataSource() datasource.DataSource {
	return &IssueTypesSchemaDataSource{}
}

func (d *IssueTypesSchemaDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "jira_issue_types_schema"
}

func (d *IssueTypesSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Jira issue type scheme by name, so a project can refer to it without hardcoding its ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Scheme ID, reported by Jira as a string for this scheme type.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Exact scheme name to look up, for example `Default Issue Type Scheme`.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Scheme description as stored in Jira.",
			},
			"self": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical API URL of the scheme.",
			},
		},
	}
}

func (d *IssueTypesSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, resp)
}

func (d *IssueTypesSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state IssueTypesSchemaModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cache, err := d.client.IssueTypes.GetAll(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to load issue type schemas", err.Error())
		return
	}

	its, ok := cache[state.Name.ValueString()]
	if !ok {
		resp.Diagnostics.AddError(
			"Issue type schema not found",
			fmt.Sprintf("No issue type schema with name %q", state.Name.ValueString()),
		)
		return
	}

	state.ID = types.StringValue(its.ID)
	state.Self = types.StringValue(its.Self)
	state.Description = types.StringValue(its.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
