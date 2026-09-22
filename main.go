package main

import (
	"context"
	"flag"
	"log"

	"github.com/bra1nles5/terraform-provider-jira-dc/src"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set at build time via -ldflags "-X main.version=x.y.z".
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run provider with debug mode")
	flag.Parse()

	err := providerserver.Serve(context.Background(), func() provider.Provider {
		return src.NewProvider(version)
	}, providerserver.ServeOpts{
		Address: "registry.terraform.io/bra1nles5/jira-dc",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
