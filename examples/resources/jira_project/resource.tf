data "jira_issue_types_schema" "default" {
  name = "Default Issue Type Scheme"
}

data "jira_priority_types_schema" "default" {
  name = "Default Priority Scheme"
}

data "jira_permission_schema" "default" {
  name = "Default Permission Scheme"
}

data "jira_workflow_schema" "default" {
  id = 10001
}

resource "jira_project" "example" {
  key         = "EX"
  name        = "Example Project"
  description = "Managed by Terraform"
  type        = "software"
  lead        = "jsmith"
  archived    = false

  issue_type_schema_id = data.jira_issue_types_schema.default.id
  priority_schema_id   = data.jira_priority_types_schema.default.id
  permission_schema_id = data.jira_permission_schema.default.id
  workflow_schema_id   = data.jira_workflow_schema.default.id
}
