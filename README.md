# Terraform Provider — Jira Data Center

[![CI](https://github.com/bra1nles5/terraform-provider-jira-dc/actions/workflows/ci.yml/badge.svg)](https://github.com/bra1nles5/terraform-provider-jira-dc/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/bra1nles5/terraform-provider-jira-dc)](https://github.com/bra1nles5/terraform-provider-jira-dc/releases/latest)
[![Terraform Registry](https://img.shields.io/badge/registry-bra1nles5%2Fjira--dc-7B42BC?logo=terraform)](https://registry.terraform.io/providers/bra1nles5/jira-dc/latest)
[![License](https://img.shields.io/github/license/bra1nles5/terraform-provider-jira-dc)](LICENSE)

Terraform provider for [Jira Data Center](https://www.atlassian.com/software/jira/data-center): manages
projects and looks up the issue type, priority, workflow and permission schemes they are built from.
Jira Cloud is not supported.

## Requirements

- Terraform ≥ 1.5
- Jira Data Center with REST API access and a Jira administrator account
  (the ScriptRunner endpoints are restricted to `jira-administrators`)
- [ScriptRunner for Jira](https://scriptrunner.adaptavist.com/) with the endpoints from
  [`jira-endpoints/`](jira-endpoints) deployed — see the [ScriptRunner setup](docs/guides/scriptrunner-setup.md) guide
- Go ≥ 1.27, only to build from source

## Quick start

```hcl
terraform {
  required_providers {
    jira = {
      source  = "bra1nles5/jira-dc"
      version = "0.1.3"
    }
  }
}

# username and token may be omitted when JIRA_USERNAME and JIRA_TOKEN are set.
provider "jira" {
  base_url = "https://jira.example.com/"
  username = "admin"
  token    = "my-api-token"
}

data "jira_issue_types_schema" "default" {
  name = "Default Issue Type Scheme"
}

data "jira_workflow_schema" "default" {
  id = 10001
}

data "jira_permission_schema" "default" {
  name = "Default Permission Scheme"
}

resource "jira_project" "example" {
  key  = "EX"
  name = "Example Project"
  type = "software"
  lead = "jsmith"

  issue_type_schema_id = data.jira_issue_types_schema.default.id
  workflow_schema_id   = data.jira_workflow_schema.default.id
  # Needed at creation: without it not even an administrator can see the new project.
  permission_schema_id = data.jira_permission_schema.default.id
}
```

Existing projects are imported by key: `terraform import jira_project.example EX`.

## Documentation

The full reference lives on the [Terraform Registry](https://registry.terraform.io/providers/bra1nles5/jira-dc/latest/docs)
and is generated from the provider schema into [`docs/`](docs):

- [Provider configuration](docs/index.md)
- Resource: [`jira_project`](docs/resources/project.md)
- Data sources: [`jira_issue_types_schema`](docs/data-sources/issue_types_schema.md),
  [`jira_priority_types_schema`](docs/data-sources/priority_types_schema.md),
  [`jira_workflow_schema`](docs/data-sources/workflow_schema.md),
  [`jira_permission_schema`](docs/data-sources/permission_schema.md)
- Guide: [ScriptRunner setup](docs/guides/scriptrunner-setup.md)

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

# Run tests. Resource tests drive real Terraform against a fake Jira: Terraform is
# taken from PATH, or the latest release is downloaded.
make unit

# Format and lint
make fmt
make lint

# Regenerate docs/ from the provider schema, examples/ and templates/
make docs

# Remove build artifacts and local state
make clean
```

The binary is installed to:
```
~/.terraform.d/plugins/registry.terraform.io/bra1nles5/jira-dc/0.1.3/<OS>_<ARCH>/
```

## License

Licensed under the Apache License, Version 2.0 — see [LICENSE](LICENSE).

Copyright 2026 bra1nles5
