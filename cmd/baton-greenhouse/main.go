package main

import (
	"context"

	cfg "github.com/conductorone/baton-greenhouse/pkg/config"
	"github.com/conductorone/baton-greenhouse/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
)

var version = "dev"

func main() {
	ctx := context.Background()
	config.RunConnector(
		ctx,
		"baton-greenhouse",
		version,
		cfg.Configuration,
		connector.New,
		connectorrunner.WithProvisioningEnabled(),
		connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2(&connector.Connector{}),
	)
}
