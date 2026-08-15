---
page_title: "Provisioning Cloud Servers"
description: |-
  Bring up a VM at Hetzner, DigitalOcean or Vultr and register it with Coolify in one
  apply.
---

# Provisioning Cloud Servers

`coolify_cloud_server` provisions a VM at Hetzner, DigitalOcean or Vultr *and*
registers it as a Coolify server in the same `apply` — the same flow as Coolify's
**Servers → New → Hetzner/DigitalOcean/Vultr** wizard, but declarative.

This is the one part of the provider that creates billable infrastructure outside
Coolify itself. Read the [Destroy behavior](#destroy-behavior) section before running
this in anything other than a sandbox.

## The three pieces

**1. A cloud provider token.** Create one at the provider (Hetzner Cloud Console →
Security → API Tokens; DigitalOcean → API → Tokens; Vultr → Account → API), then store
it in Coolify:

```terraform
resource "coolify_cloud_token" "hetzner" {
  name          = "hetzner-prod"
  provider_name = "hetzner"
  token         = var.hetzner_api_token
}
```

**2. A private key for SSH access.** Coolify connects to the new VM over SSH — the same
key used for [`coolify_server`](server.html.md):

```terraform
resource "coolify_private_key" "deploy" {
  name        = "cloud-deploy-key"
  private_key = file("~/.ssh/id_ed25519")
}
```

**3. The server itself.** Provider-specific attributes only apply to their provider —
setting Hetzner's `location`/`server_type` while `provider_name = "digitalocean"` does
nothing:

```terraform
resource "coolify_cloud_server" "worker" {
  provider_name    = "hetzner"
  cloud_token_uuid = coolify_cloud_token.hetzner.uuid
  private_key_uuid = coolify_private_key.deploy.uuid

  name             = "worker-02"
  location         = "fsn1"
  server_type      = "cx32"
  hetzner_image_id = 114690389 # ubuntu-24.04
}
```

## Discovering valid values

Locations, server types and images are provider catalogs that change over time —
`coolify_cloud_catalog` proxies them live instead of hardcoding a list that would go
stale:

```terraform
data "coolify_cloud_catalog" "hetzner_locations" {
  provider_name    = "hetzner"
  section          = "locations"
  cloud_token_uuid = coolify_cloud_token.hetzner.uuid
}

output "available_locations" {
  value = data.coolify_cloud_catalog.hetzner_locations.names
}
```

Valid `section` values per provider:

| `provider_name` | Sections |
| --- | --- |
| `hetzner` | `locations`, `server-types`, `images`, `ssh-keys`, `firewalls`, `networks` |
| `digitalocean` | `regions`, `sizes`, `images`, `ssh-keys` |
| `vultr` | `regions`, `plans`, `os`, `ssh-keys` |

`data.coolify_cloud_catalog.<name>.items` carries the full raw entries (flattened to
string values) when you need more than just the name — e.g. an image's numeric id for
`hetzner_image_id`.

## Provider-specific attributes

| Attribute | Hetzner | DigitalOcean | Vultr |
| --- | :-: | :-: | :-: |
| `location` / `region` | `location` | `region` | `region` |
| Instance size | `server_type` | `size` | `plan` |
| Image | `hetzner_image_id` (integer) | `image` (string slug) | `os_id` (integer) |
| `enable_ipv4` | ✓ | — | — |
| `enable_ipv6` | ✓ | ✓ | — |
| `monitoring` | — | ✓ | — |

Hetzner and Vultr identify their image by a numeric id; DigitalOcean by a string slug
(e.g. `ubuntu-24-04-x64`) — this is a real difference in the underlying APIs, not an
inconsistency in the provider.

## Destroy behavior

**`terraform destroy` on a `coolify_cloud_server` only deregisters the server from
Coolify — it does not terminate the VM at the provider.** Coolify's provisioning API
has no corresponding "destroy the VM" endpoint to call; the machine keeps running (and
billing) until you remove it directly in the Hetzner/DigitalOcean/Vultr console, or
through that provider's own Terraform provider. The resource emits a warning to this
effect on every delete, not just the first time.

If you want VM lifecycle fully under Terraform, provision the machine with the
provider's own official provider (`hetznercloud/hcloud`,
`digitalocean/digitalocean`, `vultr/vultr`) and register it with
[`coolify_server`](server.html.md) instead — that combination gives you a real destroy.
`coolify_cloud_server` trades that away for a single resource that does both steps in
one apply, which is the better trade for throwaway or CI-provisioned infrastructure.
