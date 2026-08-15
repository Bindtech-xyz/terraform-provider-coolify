resource "coolify_s3_storage" "backups" {
  name       = "db-backups"
  endpoint   = "https://s3.eu-west-1.amazonaws.com"
  bucket     = "my-coolify-backups"
  region     = "eu-west-1"
  access_key = var.s3_access_key
  secret_key = var.s3_secret_key
}
