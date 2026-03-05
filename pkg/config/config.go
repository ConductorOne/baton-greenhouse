package config

import (
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	clientIDField = field.StringField(
		"client_id",
		field.WithDisplayName("Greenhouse Client ID"),
		field.WithDescription("The Client ID for Greenhouse Harvest API v3 OAuth2 authentication"),
		field.WithRequired(true),
	)
	clientSecretField = field.StringField(
		"client_secret",
		field.WithDisplayName("Greenhouse Client Secret"),
		field.WithDescription("The Client Secret for Greenhouse Harvest API v3 OAuth2 authentication"),
		field.WithRequired(true),
		field.WithIsSecret(true),
	)
	onBehalfOfUserIDField = field.StringField(
		"on_behalf_of_user_id",
		field.WithDisplayName("On behalf of (User ID)"),
		field.WithDescription("Greenhouse User ID of a Site Admin to act on behalf of (used as the sub claim in the JWT token)"),
	)

	// FieldRelationships defines relationships between the fields listed in
	// ConfigurationFields that can be automatically validated. For example, a
	// username and password can be required together, or an access token can be
	// marked as mutually exclusive from the username password pair.
	FieldRelationships = []field.SchemaFieldRelationship{}
)

// Config defines the configuration fields for the Greenhouse connector.
//
//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{
		clientIDField,
		clientSecretField,
		onBehalfOfUserIDField,
	},
	field.WithConnectorDisplayName("Greenhouse"),
	field.WithHelpUrl("/docs/baton/greenhouse"),
	field.WithIconUrl("/static/app-icons/greenhouse.svg"),
)

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(cfg *Greenhouse) error {
	if cfg.Client_id == "" {
		return fmt.Errorf("client_id is required for Greenhouse Harvest API v3")
	}
	if cfg.Client_secret == "" {
		return fmt.Errorf("client_secret is required for Greenhouse Harvest API v3")
	}
	return nil
}
