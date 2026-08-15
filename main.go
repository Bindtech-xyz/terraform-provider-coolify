// Terraform provider for Coolify (https://coolify.io).
//
//go:generate go tool tfplugindocs generate --provider-name coolify
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/provider"
)

// These are set at build time by GoReleaser via -ldflags.
var (
	version = "dev"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// Must match the Terraform Registry address, otherwise `terraform init`
		// will not be able to match the binary to the `required_providers` entry.
		Address: "registry.terraform.io/bindtech-xyz/coolify",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
