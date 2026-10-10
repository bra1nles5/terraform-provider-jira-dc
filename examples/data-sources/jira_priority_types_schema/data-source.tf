data "jira_priority_types_schema" "default" {
  name = "Default Priority Scheme"
}

output "priority_ids" {
  value = data.jira_priority_types_schema.default.options_ids
}
