# Terraform Provider — Jira Data Center

Custom Terraform provider for managing [Jira Data Center](https://www.atlassian.com/software/jira/data-center) resources.

## Requirements

- Terraform ≥ 1.5
- Go ≥ 1.24 (to build from source)
- Jira Data Center instance with REST API access
- [ScriptRunner for Jira](https://scriptrunner.adaptavist.com/) plugin (required for workflow scheme and issue type scheme operations — see [ScriptRunner setup](#scriptrunner-setup))

## Authentication

The provider uses **HTTP Basic Auth** with a Jira username and API token.

Credentials can be provided via provider configuration or environment variables:

| Variable        | Description              |
|-----------------|--------------------------|
| `JIRA_USERNAME` | Jira username            |
| `JIRA_TOKEN`    | Jira API token/password  |

## Provider configuration

```hcl
terraform {
  required_providers {
    jira = {
      source  = "bra1nles5/jira-dc"
      version = "0.1.0"
    }
  }
}

provider "jira" {
  base_url = "https://jira.example.com/"
  # username and token can be omitted if JIRA_USERNAME / JIRA_TOKEN env vars are set
  username = "admin"
  token    = "my-api-token"
}
```

## Resources

### `jira_project`

Creates and manages a Jira project.

```hcl
resource "jira_project" "example" {
  key         = "EX"
  name        = "Example Project"
  description = "Managed by Terraform"
  type        = "software"   # software | business | service_desk
  lead        = "jsmith"

  archived = false

  issue_type_schema_id = data.jira_issue_types_schema.default.id
  priority_schema_id   = data.jira_priority_types_schema.default.id
  workflow_schema_id   = data.jira_workflow_schema.default.id
  permission_schema_id = data.jira_permission_schema.default.id
}
```

#### Arguments

| Argument               | Type   | Required | Description                              |
|------------------------|--------|----------|------------------------------------------|
| `key`                  | string | yes      | Unique project key (e.g. `EX`)           |
| `name`                 | string | yes      | Project display name                     |
| `type`                 | string | yes      | `software`, `business`, or `service_desk`|
| `lead`                 | string | yes      | Username of the project lead             |
| `description`          | string | no       | Project description                      |
| `archived`             | bool   | no       | Whether the project is archived (computed if omitted) |
| `issue_type_schema_id` | string | yes      | ID of the issue type scheme              |
| `priority_schema_id`   | int    | no       | ID of the priority scheme (computed if omitted; null for an archived project) |
| `workflow_schema_id`   | int    | yes      | ID of the workflow scheme                |
| `permission_schema_id` | int    | no       | ID of the permission scheme (computed if omitted; null for an archived project) |

---

## Data Sources

### `jira_issue_types_schema`

Looks up an issue type scheme by name.

```hcl
data "jira_issue_types_schema" "default" {
  name = "Default Issue Type Scheme"
}

output "id" {
  value = data.jira_issue_types_schema.default.id
}
```

### `jira_priority_types_schema`

Looks up a priority scheme by name.

```hcl
data "jira_priority_types_schema" "default" {
  name = "Default Priority Schema"
}
```

### `jira_workflow_schema`

Looks up a workflow scheme by ID or by name — set exactly one of them.

```hcl
data "jira_workflow_schema" "by_id" {
  id = 10001
}

data "jira_workflow_schema" "by_name" {
  name = "My Workflow Scheme"
}
```

> **Note:** the `name` form requires the `listWorkflowSchemes` ScriptRunner endpoint — REST API v2 of Jira DC has no listing for workflow schemes. The `id` form works without it.

### `jira_permission_schema`

Looks up a permission scheme by name.

```hcl
data "jira_permission_schema" "default" {
  name = "Default Permission Scheme"
}
```

---

## Building from source

```bash
# Build and install to local Terraform plugin directory
make install

# Build only
make build

# Run terraform plan against main.tf (for manual testing).
# main.tf is git-ignored: copy the example and put your own values in it first.
#   cp main.tf.example main.tf
make test

# Run unit tests
make unit

# Format and lint
make fmt
make lint

# Remove build artifacts and local state
make clean
```

The binary is installed to:
```
~/.terraform.d/plugins/registry.terraform.io/bra1nles5/jira-dc/0.1.0/<OS>_<ARCH>/
```

---

## ScriptRunner setup

Some operations are not available in Jira DC REST API v2 and are implemented via custom [ScriptRunner](https://scriptrunner.adaptavist.com/) endpoints deployed on the Jira server. **The provider will not function without these scripts.**

### Prerequisites

- [ScriptRunner for Jira](https://marketplace.atlassian.com/apps/6820/scriptrunner-for-jira) plugin installed and licensed on your Jira Data Center instance.
- Jira administrator access to create REST endpoints.

### Deployment steps

1. Log in to Jira as an administrator.
2. Go to **Jira Administration (⚙) → Manage apps → ScriptRunner → REST Endpoints**.
3. Click **Add new endpoint**.
4. Paste the full contents of the Groovy script into the script editor.
5. Save. Repeat for each of the four scripts below.

You can verify each endpoint is working by hitting its URL with curl:

```bash
curl -u admin:token "https://jira.example.com/rest/scriptrunner/latest/custom/getWorkflowScheme?projectKey=EX"
```

### Endpoint reference

#### `listWorkflowSchemes.groovy`

| Property | Value |
|----------|-------|
| File | `jira-endpoints/listWorkflowSchemes.groovy` |
| Method | `GET` |
| Path | `/rest/scriptrunner/latest/custom/listWorkflowSchemes` |
| Required groups | `jira-administrators` |
| Query param | — |

Required by `data "jira_workflow_schema"` when it looks a scheme up by `name`: REST API v2 of Jira DC lists no workflow schemes.

**Response:**
```json
{
  "schemes": [
    { "self": "https://jira.example.com/rest/api/2/workflowscheme/10001", "id": 10001, "name": "My Workflow Scheme", "description": "" }
  ]
}
```

---

#### `getWorkflowScheme.groovy`

| Property | Value |
|----------|-------|
| File | `jira-endpoints/getWorkflowScheme.groovy` |
| Method | `GET` |
| Path | `/rest/scriptrunner/latest/custom/getWorkflowScheme` |
| Required groups | `jira-administrators` |
| Query param | `projectKey` (string) |

**Response:**
```json
{ "success": true, "projectKey": "EX", "schemeId": 10001, "schemeName": "My Workflow Scheme" }
```

---

#### `assignWorkflowScheme.groovy`

| Property | Value |
|----------|-------|
| File | `jira-endpoints/assignWorkflowScheme.groovy` |
| Method | `PUT` |
| Path | `/rest/scriptrunner/latest/custom/assignWorkflowScheme` |
| Required groups | `jira-administrators` |
| Request body | JSON (see below) |

**Request body:**
```json
{ "projectKey": "EX", "schemeId": 10001, "updateDraft": true }
```

**Response:**
```json
{ "success": true, "projectKey": "EX", "schemeId": 10001, "schemeName": "My Workflow Scheme", "message": "..." }
```

> **Note:** Setting `updateDraft: true` triggers automatic migration for projects with existing issues. This may take time on large projects.

---

#### `GetIssueTypesSchemaAssociatedToTheProject.groovy`

| Property | Value |
|----------|-------|
| File | `jira-endpoints/GetIssueTypesSchemaAssociatedToTheProject.groovy` |
| Method | `GET` |
| Path | `/rest/scriptrunner/latest/custom/getIssueTypeScheme` |
| Required groups | `jira-administrators` |
| Query param | `projectKey` (string) |

**Response:**
```json
{
  "projectKey": "EX",
  "schemeId": "10200",
  "schemeName": "Default Issue Type Scheme",
  "schemeDescription": "...",
  "defaultIssueType": "Story",
  "issueTypes": ["Epic", "Story", "Task", "Bug"]
}
```

> **Note:** `schemeId` is returned as a string by this endpoint. The provider stores it as a string in `issue_type_schema_id`.

---

## License

Licensed under the Apache License, Version 2.0 — see [LICENSE](LICENSE).

Copyright 2026 bra1nles5
