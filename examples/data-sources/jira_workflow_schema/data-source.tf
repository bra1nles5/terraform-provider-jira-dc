# By ID: works against plain REST API v2.
data "jira_workflow_schema" "by_id" {
  id = 10001
}

# By name: needs the listWorkflowSchemes ScriptRunner endpoint.
data "jira_workflow_schema" "by_name" {
  name = "My Workflow Scheme"
}
