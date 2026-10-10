---
page_title: "ScriptRunner setup"
subcategory: ""
description: |-
  Deploy the ScriptRunner REST endpoints the provider needs on your Jira Data Center instance.
---

# ScriptRunner setup

REST API v2 of Jira Data Center cannot assign a workflow scheme to a project, report a
project's issue type scheme or list workflow schemes. The provider does these through custom
[ScriptRunner](https://scriptrunner.adaptavist.com/) REST endpoints, which have to be deployed
on the Jira instance before the provider is used.

| Endpoint | Needed by |
|----------|-----------|
| `assignWorkflowScheme` | `jira_project`: always |
| `getIssueTypeScheme` | `jira_project`: always |
| `listWorkflowSchemes` | `jira_workflow_schema`: only when looking a scheme up by `name` |

## Prerequisites

- [ScriptRunner for Jira](https://marketplace.atlassian.com/apps/6820/scriptrunner-for-jira)
  installed and licensed on the Jira Data Center instance.
- Jira administrator access to create REST endpoints.

## Deployment

1. Log in to Jira as an administrator.
2. Go to **Jira Administration → Manage apps → ScriptRunner → REST Endpoints**.
3. Click **Add new endpoint**.
4. Paste the full contents of the Groovy script into the script editor.
5. Save, and repeat for each script below.

Check that an endpoint answers:

```shell
curl -u admin:token "https://jira.example.com/rest/scriptrunner/latest/custom/getIssueTypeScheme?projectKey=EX"
```

## Endpoint reference

### assignWorkflowScheme

| Property | Value |
|----------|-------|
| Script | [`assignWorkflowScheme.groovy`](https://github.com/bra1nles5/terraform-provider-jira-dc/blob/main/jira-endpoints/assignWorkflowScheme.groovy) |
| Method | `PUT` |
| Path | `/rest/scriptrunner/latest/custom/assignWorkflowScheme` |
| Required groups | `jira-administrators` |

Request body:

```json
{ "projectKey": "EX", "schemeId": 10001, "updateDraft": true }
```

Response:

```json
{ "success": true, "projectKey": "EX", "schemeId": 10001, "schemeName": "My Workflow Scheme", "message": "..." }
```

~> With `updateDraft: true` Jira migrates the existing issues of the project to the new
workflows, which may take a while on a large project.

### getIssueTypeScheme

| Property | Value |
|----------|-------|
| Script | [`GetIssueTypesSchemaAssociatedToTheProject.groovy`](https://github.com/bra1nles5/terraform-provider-jira-dc/blob/main/jira-endpoints/GetIssueTypesSchemaAssociatedToTheProject.groovy) |
| Method | `GET` |
| Path | `/rest/scriptrunner/latest/custom/getIssueTypeScheme` |
| Required groups | `jira-administrators` |
| Query parameter | `projectKey` |

Response:

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

-> `schemeId` comes back as a string, which is why `issue_type_schema_id` is a string too.

### listWorkflowSchemes

| Property | Value |
|----------|-------|
| Script | [`listWorkflowSchemes.groovy`](https://github.com/bra1nles5/terraform-provider-jira-dc/blob/main/jira-endpoints/listWorkflowSchemes.groovy) |
| Method | `GET` |
| Path | `/rest/scriptrunner/latest/custom/listWorkflowSchemes` |
| Required groups | `jira-administrators` |

Response:

```json
{
  "schemes": [
    { "self": "https://jira.example.com/rest/api/2/workflowscheme/10001", "id": 10001, "name": "My Workflow Scheme", "description": "" }
  ]
}
```
