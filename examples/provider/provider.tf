terraform {
  required_providers {
    coolify = {
      source  = "d3nailabs/coolify"
      version = "~> 0.1"
    }
  }
}

# Token comes from the COOLIFY_TOKEN environment variable.
provider "coolify" {
  endpoint = "https://coolify.example.com"
}
