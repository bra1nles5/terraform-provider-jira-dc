package datasources

import (
	"context"
	"fmt"
	jira "github.com/bra1nles5/terraform-provider-jira-dc/src/jiraClient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PermissionSchemaDataSource struct {
	client *jira.JiraClient
}

type PermissionSchemaModel struct {
	ID          types.Int32  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Self        types.String `tfsdk:"self"`
}

func NewPermissionSchemaDataSource() datasource.DataSource {
	return &PermissionSchemaDataSource{}
}

func (d *PermissionSchemaDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "jira_permission_schema"
}

func (d *PermissionSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Jira permission scheme by name, so a project can refer to it without hardcoding its ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Computed:    true,
				Description: "Scheme ID.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Exact scheme name to look up, for example `Default Permission Scheme`.",
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

func (d *PermissionSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, resp)
}

func (d *PermissionSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state PermissionSchemaModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cache, err := d.client.Permissions.GetAll(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to load permission schemas", err.Error())
		return
	}

	its, ok := cache[state.Name.ValueString()]
	if !ok {
		resp.Diagnostics.AddError(
			"Permission schema not found",
			fmt.Sprintf("No permission schema with name %q", state.Name.ValueString()),
		)
		return
	}

	state.ID = types.Int32Value(its.ID)
	state.Self = types.StringValue(its.Self)
	state.Description = types.StringValue(its.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
