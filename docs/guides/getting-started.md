---
page_title: "Getting Started with the Coolify Provider"
description: |-
  Install the provider, configure it, and deploy your first application end to end.
---

# Getting Started

This walks through a complete first deployment: install the provider, configure it,
register a server, and deploy an application from a public git repository. It assumes
you already have a running Coolify instance (cloud or self-hosted) with at least one
server connected — either the built-in `localhost` server or one added through the
Coolify UI.

## 1. Create an API token

In the Coolify dashboard, go to **Keys & Tokens → API tokens** and create a token with
at least the `write` ability. Add `read:sensitive` if you want generated database
credentials and secret values readable back into Terraform state — see
[Token Abilities & Sensitive Data](authentication-and-token-abilities.html.md) for the
trade-off.

Export it rather than writing it into configuration:

```sh
export COOLIFY_TOKEN="1|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

If you self-host, also export the instance URL — otherwise the provider defaults to
Coolify Cloud:

```sh
export COOLIFY_ENDPOINT="https://coolify.example.com"
```

## 2. Declare the provider

```terraform
terraform {
  required_providers {
    coolify = {
      source  = "d3nailabs/coolify"
      version = "~> 0.1"
    }
  }
}

provider "coolify" {}
```

`terraform init` downloads the provider. On `terraform plan` or `apply`, `Configure`
calls `GET /version` to fail fast with a clear error if the endpoint or token is wrong
— you will never get halfway through an apply before discovering a typo'd URL.

## 3. Find a server to deploy on

Every Coolify instance has at least one server as soon as it is installed. Look it up
instead of hardcoding its UUID:

```terraform
data "coolify_servers" "all" {}

locals {
  # The built-in server is usually named "localhost".
  server_uuid = one([for s in data.coolify_servers.all.servers : s.uuid if s.name == "localhost"])
}
```

## 4. Create a project and deploy an application

Coolify organizes resources as project → environment → workload. A project always
starts with a `production` environment, so referencing it by name needs no extra
resource:

```terraform
resource "coolify_project" "demo" {
  name = "getting-started"
}

resource "coolify_application" "web" {
  project_uuid      = coolify_project.demo.uuid
  environment_name  = "production"
  server_uuid       = local.server_uuid

  git_repository = "https://github.com/coollabsio/coolify-examples"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"

  instant_deploy = true
}

output "app_url" {
  value = coolify_application.web.fqdn
}
```

`instant_deploy = true` queues a build right after the resource is created — the same
as clicking **Deploy** in the UI. Leave it `false` (the default) if you would rather
trigger the first deployment separately, e.g. via `coolify_resource_action`.

## 5. Apply

```sh
terraform init
terraform plan
terraform apply
```

Once applied, `terraform output app_url` prints the domain Coolify assigned (or the one
you set explicitly via `domains`).

## Next steps

- [Deploying a Full Stack](full-stack-deployment.html.md) wires a database and
  environment variables into the same application, and shows the `for_each` pattern for
  deploying many applications from one block.
- [Importing Existing Infrastructure](importing-existing-infrastructure.html.md) brings
  resources created through the Coolify UI under Terraform management.
