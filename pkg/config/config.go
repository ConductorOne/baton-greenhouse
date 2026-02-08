package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	usernameField = field.StringField(
		"username",
		field.WithDisplayName("Greenhouse username (API token)"),
		field.WithDescription("The username is your Greenhouse API token"),
		field.WithIsSecret(true),
	)
	onBehalfOfField = field.StringField(
		"on_behalf_of_email",
		field.WithDisplayName("On behalf of"),
		field.WithDescription("Email of the Site Admin user"),
	)
	BaseURLField = field.StringField(
		"base-url",
		field.WithDescription("Override the Greenhouse API URL (for testing)"),
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
		usernameField,
		onBehalfOfField,
		BaseURLField,
	},
	field.WithConnectorDisplayName("Greenhouse"),
	field.WithHelpUrl("/docs/baton/greenhouse"),
	field.WithIconUrl("/static/app-icons/greenhouse.svg"),
)

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(_ *Greenhouse) error {
	return nil
}
