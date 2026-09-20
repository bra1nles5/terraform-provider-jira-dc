# Changelog

All notable changes to this provider are documented here.
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.1.0

First release.

### Added

- `jira_project` resource: create, read, update and delete Jira projects,
  with `terraform import` support and archive/restore handling.
- Data sources for looking up the schemes a project is built from:
  `jira_issue_types_schema`, `jira_priority_types_schema`,
  `jira_workflow_schema` (by id or by name) and `jira_permission_schema`.
- Jira client with Basic Auth, a lazy scheme cache, retries on 429 (honouring
  `Retry-After`) and 5xx (exponential back-off), and typed 404/403 errors.
- ScriptRunner endpoints under `jira-endpoints/` for the operations Jira Data
  Center REST API v2 does not expose: workflow scheme assignment and lookup,
  issue type scheme lookup and workflow scheme listing.

### Notes

- The provider targets Jira **Data Center**, not Jira Cloud.
- The ScriptRunner endpoints must be deployed on the Jira instance before use;
  see the README for the deployment steps.
