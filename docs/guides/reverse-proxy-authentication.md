---
page_title: "Reaching Coolify Behind an Authenticating Edge"
description: |-
  Use the provider's headers attribute to reach a Coolify instance sitting behind
  Cloudflare Access or a similar edge auth layer.
---

# Reaching Coolify Behind an Authenticating Edge

Some Coolify instances are not directly reachable — they sit behind an edge layer that
authenticates the request *before* it ever reaches Coolify: a Cloudflare Access
application, an `oauth2-proxy` sidecar, a header-based mTLS gateway, an internal API
gateway. Coolify's own bearer token has no say over that layer; it only governs access
once the request arrives at Coolify itself. Reaching the API therefore needs the edge
layer's own credential, sent as one or more HTTP headers, on every request.

The `headers` provider attribute exists for exactly this, and only this — it is a flat
map of header name to value, applied to every request before the provider's own
`Authorization: Bearer <token>` header (which cannot be overridden through it, even by
accident: `headers = { Authorization = "..." }` has no effect). The provider does not
know or care what edge layer, if any, sits in front — it just forwards headers.

## Worked example: Cloudflare Access

Coolify behind a [Cloudflare Access](https://developers.cloudflare.com/cloudflare-one/policies/access/)
application requires a [service token](https://developers.cloudflare.com/cloudflare-one/policies/access/service-tokens/)
for non-browser clients like Terraform — a human SSO login has nothing to authenticate
against here. Create one in **Zero Trust → Access → Service Auth → Service Tokens**,
then scope an Access policy to allow it — **on a path-specific application, not the one
protecting the dashboard**, so the service token only ever unlocks the API, never a
human-facing login bypass:

- Application: hostname `coolify.example.com`, path `/api/*`
- Policy: type **Service Auth**, action **Allow**, select the service token

Cloudflare Access can also be provisioned as code with the
[`cloudflare` provider](https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/resources/zero_trust_access_service_token) —
`cloudflare_zero_trust_access_service_token` creates the token, and its `client_id` /
`client_secret` attributes feed straight into this provider's `headers`:

```terraform
resource "cloudflare_zero_trust_access_service_token" "coolify_api" {
  account_id = var.cloudflare_account_id
  name       = "terraform-provider-coolify"
}

provider "coolify" {
  endpoint = "https://coolify.example.com"
  # token via COOLIFY_TOKEN, as usual

  headers = {
    "CF-Access-Client-Id"     = cloudflare_zero_trust_access_service_token.coolify_api.client_id
    "CF-Access-Client-Secret" = cloudflare_zero_trust_access_service_token.coolify_api.client_secret
  }
}
```

## The chicken-and-egg trap, and why it usually isn't one

A provider block is configured once, before Terraform plans *any* resource — so it
cannot depend on a resource this same configuration also creates: `headers` built from
`cloudflare_zero_trust_access_service_token.coolify_api.client_id` is `(known after
apply)` on the very first `plan`, and `Configure` fails with a clear
"Unknown Coolify provider headers" error rather than an opaque connectivity failure.

In practice this resolves itself after the first apply that creates the token: once
`cloudflare_zero_trust_access_service_token.coolify_api` exists in state, its
`client_id`/`client_secret` are known values on every subsequent plan, and the
`coolify` provider configures normally — the token and the Coolify resources that need
it can live in the same configuration going forward. Only the very first apply needs
the service token resource targeted on its own:

```sh
terraform apply -target=cloudflare_zero_trust_access_service_token.coolify_api
terraform apply
```

If you would rather never deal with this even once, provision the service token in a
separate Terraform configuration/state (or through the Cloudflare dashboard directly)
and pass its `client_id`/`client_secret` in as plain variables instead of resource
references.

## Verifying the edge, not just the API

Once headers are wired up, a cheap sanity check distinguishes "the edge let the
request through" from "the edge is still blocking it": the dashboard's root path
(everything *outside* your API application's path scope) should still redirect to
Access's login — if it stops doing that, your Access policy is broader than intended
and the dashboard itself is now unauthenticated.

```sh
curl -sI https://coolify.example.com/ | grep -i location   # should redirect to Access login
curl -s  https://coolify.example.com/api/v1/version \
  -H "CF-Access-Client-Id: $CF_ACCESS_CLIENT_ID" \
  -H "CF-Access-Client-Secret: $CF_ACCESS_CLIENT_SECRET" \
  -H "Authorization: Bearer $COOLIFY_TOKEN"                # should return a version string
```
