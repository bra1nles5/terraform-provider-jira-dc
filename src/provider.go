package src

import (
	"context"
	"os"
	"github.com/bra1nles5/terraform-provider-jira-dc/src/dataSources"
	"github.com/bra1nles5/terraform-provider-jira-dc/src/jiraClient"
	"github.com/bra1nles5/terraform-provider-jira-dc/src/resources"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type jiraProviderModel struct {
	BaseURL  types.String `tfsdk:"base_url"`
	Username types.String `tfsdk:"username"`
	Token    types.String `tfsdk:"token"`
}

type JiraDataCenterProvider struct{}

func NewProvider() provider.Provider {
	return &JiraDataCenterProvider{}
}

func (p *JiraDataCenterProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jira"
}

func (p *JiraDataCenterProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{
		Attributes: map[string]providerschema.Attribute{
			"base_url": providerschema.StringAttribute{
				Required:    true,
				Description: "Base URL of the Jira Data Center instance, for example https://jira.example.com",
			},
			"username": providerschema.StringAttribute{
				Optional:    true,
				Description: "Jira username for Basic Auth. Can be set via the JIRA_USERNAME environment variable.",
			},
			"token": providerschema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "Jira API token. Can be set via the JIRA_TOKEN environment variable.",
				MarkdownDescription: "Jira API token. Can be set via the `JIRA_TOKEN` environment variable.",
			},
		},
	}
}

func (p *JiraDataCenterProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config jiraProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.BaseURL.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Jira base_url",
			"The provider needs `base_url` to reach Jira.",
		)
		return
	}

	username := config.Username.ValueString()
	if username == "" {
		username = os.Getenv("JIRA_USERNAME")
	}

	token := config.Token.ValueString()
	if token == "" {
		token = os.Getenv("JIRA_TOKEN")
	}

	client := jira.NewJiraClient(config.BaseURL.ValueString(), username, token)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *JiraDataCenterProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewProjectResource,
	}
}

func (p *JiraDataCenterProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewIssueTypesSchemaDataSource,
		datasources.NewPriorityTypesSchemaDataSource,
		datasources.NewWorkflowSchemaDataSource,
		datasources.NewPermissionSchemaDataSource,
	}
}
