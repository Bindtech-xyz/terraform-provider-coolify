# Requires a root-team (team 0) API token. Enables the MCP server without
# touching the REST API itself (api_enabled defaults to true).
resource "coolify_api_settings" "instance" {
  mcp_enabled = true
}
