terraform {
  required_providers {
    jira = {
      source  = "bra1nles5/jira-dc"
      version = "0.1.3"
    }
  }
}

provider "jira" {
  base_url = "https://jira.example.com/"

  # Both may be omitted when JIRA_USERNAME and JIRA_TOKEN are set.
  username = "admin"
  token    = "my-api-token"
}
