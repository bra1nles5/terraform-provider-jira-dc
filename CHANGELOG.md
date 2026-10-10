# Changelog

All notable changes to this provider are documented here.
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Fixed

- Archived projects can be imported and managed. Jira hides their workflow,
  priority and permission schemes, so after an import those were empty in state,
  every plan wanted to add them, and apply failed with 403 halfway, having already
  changed the issue type scheme. Now, while a project stays archived:
  - a hidden scheme that state knows nothing about takes the configured value
    without calling Jira, with a warning;
  - any change Jira refuses (name, description, lead, type, a known hidden
    scheme) fails before the first call, so the project is never left half-changed;
  - the issue type scheme can still change, as Jira accepts it.
- Restoring a project reads its schemes back from Jira first, so values recorded
  unchecked while it was archived are compared and fixed in the same apply.

### Changed

- `workflow_schema_id` is optional, like `priority_schema_id` and
  `permission_schema_id`. Left unset, the project keeps the scheme Jira gave it.
  Configurations generated with `-generate-config-out` for an archived project
  no longer fail with "Missing Configuration for Required Attribute".

### Security

- Release binaries are built with Go 1.27.2 instead of 1.24.0, which no longer
  gets security fixes; `golang.org/x/net`, `golang.org/x/text` and
  `google.golang.org/grpc` are updated past their known vulnerabilities.
  `govulncheck` reports none.
- Building from source now needs Go 1.27.

### Documentation

- Registry pages show usage examples for the provider, `jira_project` and every
  data source, how to import a project, and a provider description.
- New "ScriptRunner setup" guide in the Registry: which endpoints the provider
  needs, how to deploy them and what they answer.
- README is now an overview with a quick start; the argument reference and the
  ScriptRunner endpoint reference live only in `docs/`, so the two cannot drift.

## v0.1.3

### Fixed

- `jira_project` can be restored from the archive: restore now calls
  `PUT /rest/api/2/project/{key}/restore` instead of `DELETE .../archive`, which
  Jira Data Center answers with 405. Setting `archived = false` on an archived
  project used to fail.
- `getIssueTypeScheme` ScriptRunner endpoint lists the issue types of the
  project's scheme; it returned `[null]` for a global scheme. Redeploy the script.

### Changed

- The workflow scheme of a project is read through REST API v2
  (`GET /rest/api/2/project/{key}/workflowscheme`). The `getWorkflowScheme`
  ScriptRunner endpoint is no longer needed and has been removed; three
  endpoints remain.
- Schemes are verified before a project created with `archived = true` is
  archived, and an unreadable workflow scheme of an archived project keeps its
  value in state instead of failing the refresh.

## v0.1.2

### Fixed

- `jira_project` no longer fails with "Provider produced inconsistent result after
  apply" when `lead` differs from Jira's spelling only in case. Jira Data Center user
  names are case-insensitive: the configured spelling is kept, and plans no longer
  show a change to `lead` after every refresh.

## v0.1.1

### Fixed

- `jira_project` delete no longer reports success when Jira refuses it (400, 401,
  403, 5xx); the project used to leave state while staying in Jira. A project that
  is already gone (404) still counts as deleted.
- `jira_project` update restores an archived project before applying other
  changes instead of after them, so those changes do not hit an archived project.

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
