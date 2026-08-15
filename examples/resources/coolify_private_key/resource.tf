resource "coolify_private_key" "deploy" {
  name        = "deploy-key"
  private_key = file("~/.ssh/id_ed25519")
}
