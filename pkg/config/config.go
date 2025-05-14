package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	usernameField   = field.StringField("username", field.WithRequired(true), field.WithDescription("The username is your Greenhouse API token"))
	onBehalfOfField = field.StringField("on_behalf_of_email", field.WithRequired(false), field.WithDescription("Email of the Site Admin user"))

	// FieldRelationships defines relationships between the fields listed in
	// ConfigurationFields that can be automatically validated. For example, a
	// username and password can be required together, or an access token can be
	// marked as mutually exclusive from the username password pair.
	FieldRelationships = []field.SchemaFieldRelationship{}
)

//go:generate go run ./gen
var Config = field.NewConfiguration([]field.SchemaField{
	usernameField,
	onBehalfOfField,
})

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(ghc *Greenhouse) error {
	return nil
}
