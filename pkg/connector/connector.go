// Package connector implements the Greenhouse connector for baton.
package connector

import (
	"context"
	"fmt"
	"io"

	"github.com/conductorone/baton-greenhouse/pkg/client"
	cfgpkg "github.com/conductorone/baton-greenhouse/pkg/config"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
)

// Connector implements the baton connector interface for Greenhouse.
type Connector struct {
	client *client.GreenhouseClient
}

// ResourceSyncers returns a ResourceSyncer for each resource type that should be synced from the upstream service.
func (d *Connector) ResourceSyncers(_ context.Context) []connectorbuilder.ResourceSyncerV2 {
	return []connectorbuilder.ResourceSyncerV2{
		newUserBuilder(d.client),
		newRoleBuilder(d.client),
	}
}

// Asset takes an input AssetRef and attempts to fetch it using the connector's authenticated http client
// It streams a response, always starting with a metadata object, following by chunked payloads for the asset.
func (d *Connector) Asset(_ context.Context, _ *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata returns metadata about the connector.
func (d *Connector) Metadata(_ context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Greenhouse Connector",
		Description: "Connector to sync users to Greenhouse",
		AccountCreationSchema: &v2.ConnectorAccountCreationSchema{
			FieldMap: map[string]*v2.ConnectorAccountCreationSchema_Field{
				"email": {
					DisplayName: "Email",
					Required:    true,
					Description: "Email address of the user",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "user@example.com",
					Order:       1,
				},
				"first_name": {
					DisplayName: "First Name",
					Required:    true,
					Description: "User's first name",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "John",
					Order:       2,
				},
				"last_name": {
					DisplayName: "Last Name",
					Required:    true,
					Description: "User's last name",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "Travolta",
					Order:       3,
				},
			},
		},
	}, nil
}

// Validate is called to ensure that the connector is properly configured. It should exercise any API credentials
// to be sure that they are valid.
func (d *Connector) Validate(_ context.Context) (annotations.Annotations, error) {
	return nil, nil
}

// New returns a new instance of the connector.
// New returns a new instance of the connector.
//
// The *cli.ConnectorOpts parameter is part of the V2 entrypoint contract; it
// carries runtime options such as the sync resource-type filter. It is accepted
// but not yet read here.
func New(ctx context.Context, cfg *cfgpkg.Greenhouse, _ *cli.ConnectorOpts) (connectorbuilder.ConnectorBuilderV2, []connectorbuilder.Opt, error) {
	c, err := client.New(ctx, cfg.Username, cfg.On_behalf_of_email, cfg.BaseUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create a connector, error: %w", err)
	}
	return &Connector{
		client: c,
	}, nil, nil
}
