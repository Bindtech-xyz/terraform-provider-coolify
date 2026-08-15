resource "coolify_cloud_init_script" "base" {
  name   = "base-hardening"
  script = <<-EOT
    #cloud-config
    package_update: true
    packages: [fail2ban]
  EOT
}
