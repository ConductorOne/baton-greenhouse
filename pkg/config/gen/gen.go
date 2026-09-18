// Package main generates configuration code for the Greenhouse connector.
package main

import (
	cfg "github.com/conductorone/baton-greenhouse/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("greenhouse", cfg.Configuration)
}
