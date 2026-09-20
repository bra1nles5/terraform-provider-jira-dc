package datasources

import (
	"fmt"
	jira "github.com/bra1nles5/terraform-provider-jira-dc/src/jiraClient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func configureClient(providerData any, resp *datasource.ConfigureResponse) *jira.JiraClient {
	if providerData == nil {
		return nil
	}
	client, ok := providerData.(*jira.JiraClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			fmt.Sprintf("Expected *JiraClient, got: %T", providerData),
		)
		return nil
	}
	return client
}
