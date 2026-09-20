package datasources

import (
	"context"
	"fmt"
	jira "github.com/bra1nles5/terraform-provider-jira-dc/src/jiraClient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IssuePrioritySchemaDataSource struct {
	client *jira.JiraClient
}

type IssuePrioritySchemaModel struct {
	ID          types.Int32    `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	OptionsIds  []types.String `tfsdk:"options_ids"`
	Self        types.String   `tfsdk:"self"`
}

func NewPriorityTypesSchemaDataSource() datasource.DataSource {
	return &IssuePrioritySchemaDataSource{}
}

func (d *IssuePrioritySchemaDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "jira_priority_types_schema"
}

func (d *IssuePrioritySchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Jira priority scheme by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Computed:    true,
				Description: "Uniq id.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Schemas name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Issue Priority Schema Description.",
			},
			"self": schema.StringAttribute{
				Computed:    true,
				Description: "Issue Priority Schema link.",
			},
			"options_ids": schema.ListAttribute{
				Computed:    true,
				Description: "Options Ids.",
				ElementType: types.StringType,
			},
		},
	}
}

func (d *IssuePrioritySchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, resp)
}

func (d *IssuePrioritySchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state IssuePrioritySchemaModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cache, err := d.client.Priorities.GetAll(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to load priority schemas", err.Error())
		return
	}

	its, ok := cache[state.Name.ValueString()]
	if !ok {
		resp.Diagnostics.AddError(
			"Priority schema not found",
			fmt.Sprintf("No priority schema with name %q", state.Name.ValueString()),
		)
		return
	}

	state.ID = types.Int32Value(its.ID)
	state.Self = types.StringValue(its.Self)
	state.Description = types.StringValue(its.Description)

	var oids []types.String
	for _, id := range its.OptionIds {
		oids = append(oids, types.StringValue(id))
	}
	state.OptionsIds = oids
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
