data "coolify_instance" "this" {}

output "coolify_version" {
  value = data.coolify_instance.this.version
}
