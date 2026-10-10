data "jira_issue_types_schema" "default" {
  name = "Default Issue Type Scheme"
}

output "issue_type_schema_id" {
  value = data.jira_issue_types_schema.default.id
}
