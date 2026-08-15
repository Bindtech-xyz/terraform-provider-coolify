---
page_title: "Token Abilities & Sensitive Data"
description: |-
  What each Coolify API token ability unlocks, and how this provider behaves with and
  without read:sensitive.
---

# Token Abilities & Sensitive Data

Coolify API tokens are scoped by **ability**, granted when the token is created under
**Keys & Tokens → API tokens**. This guide is the detailed version of the table on the
[provider overview](../index.html.md#authentication).

## The three abilities

| Ability | What it gates |
| --- | --- |
| `read` | `GET` endpoints — every data source, and the `Read` step of every resource (state refresh). |
| `write` | `POST`/`PATCH`/`DELETE` endpoints — required for `Create`, `Update` and `Delete` on every resource this provider manages. Without it, `terraform apply` fails on the first resource it tries to create. |
| `read:sensitive` | An additional gate *on top of* `read`, applied per-field rather than per-endpoint. Fields Coolify considers sensitive — generated database passwords, private key material, S3 credentials, environment variable values — are present in the JSON response only when the token carries this ability; otherwise the API returns them as empty strings. |

A token needs `write` to be usable at all with this provider. `read:sensitive` is
optional and changes provider *behavior*, not capability — every resource still applies
correctly without it.

## What happens without `read:sensitive`

Take `coolify_database`: create it with no `postgres_password` set, and Coolify
generates one. On every subsequent `Read` (state refresh, or right after `Create`), the
API returns `"postgres_password": ""` for a token without `read:sensitive` — not
because the value doesn't exist, but because the API is hiding it.

A naive implementation would take that empty string at face value and either:

- overwrite the previously-known value with `""`, corrupting anything downstream that
  references it, or
- show a permanent diff on every plan (`"" → (known after apply)` forever), since the
  configured/prior value never matches what the API just said.

This provider does neither. Every sensitive attribute is merged with a rule along the
lines of: *if the API says empty and the prior state had a value, keep the prior
value*. Concretely (`internal/provider/helpers.go`, `keepPriorIfHidden`):

```go
func keepPriorIfHidden(apiValue string, prior types.String) types.String {
	if apiValue == "" && !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	return types.StringValue(apiValue)
}
```

This is why the provider quietly works with a `write`-only token: it trusts its own
prior knowledge of a secret over an API response it knows is deliberately redacted.

## The trade-off: drift detection

The cost of that behavior is precise: **a `write`-only token cannot detect drift on
sensitive fields.** If you rotate a database password directly in the Coolify UI (or
some other tool does), this provider has no way to distinguish "the API is hiding an
unchanged value" from "the value changed and the API is hiding the new one" — both look
identical (an empty string) on the wire. It will keep showing the old value in state
indefinitely, and `terraform plan` will report no changes.

With `read:sensitive`, the real value comes back on every read, so out-of-band rotation
shows up as an ordinary diff like any other attribute.

## Choosing an ability set

| Scenario | Recommended abilities |
| --- | --- |
| CI pipeline that only applies configuration written in `.tf` files, secrets never touched outside Terraform | `write` |
| Local/interactive use, want `terraform plan` to ever show a Coolify-generated password | `write`, `read:sensitive` |
| Read-only tooling (dashboards, drift reports) built on this provider's data sources | `read` |

There is no ability that grants `read:sensitive` without `read` — it is additive, never
a substitute.

## Affected resources

Every resource with attributes hidden without `read:sensitive`:

- `coolify_database` — all engine-specific password/user attributes, `internal_db_url`,
  `external_db_url`
- `coolify_private_key.private_key`
- `coolify_environment_variable.value`, `coolify_shared_environment_variable.value`
- `coolify_s3_storage.access_key` / `.secret_key`
- `coolify_github_app.client_id`, `coolify_gitlab_app.client_id`
- `coolify_cloud_token.token`
- `coolify_cloud_init_script.script`
- `coolify_notification_settings` — webhook URLs, SMTP password, API keys/tokens
