![Baton Logo](./docs/images/baton-logo.png)

# `baton-greenhouse` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-greenhouse.svg)](https://pkg.go.dev/github.com/conductorone/baton-greenhouse) ![main ci](https://github.com/conductorone/baton-greenhouse/actions/workflows/main.yaml/badge.svg)

`baton-greenhouse` is a connector for [Greenhouse](https://www.greenhouse.com/) built using the [Baton SDK](https://github.com/conductorone/baton-sdk).

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

To use this connector you need to provide an API key from the Harvest API. Go to Configure > Dev Center > API Credentials and make a new API Key.

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-greenhouse
baton-greenhouse --username=API_TOKEN
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_USERNAME=harvest-api-key -e BATON_DOMAIN_URL=domain_url -e BATON_API_KEY=apiKey -e BATON_USERNAME=username ghcr.io/conductorone/baton-greenhouse:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-greenhouse/cmd/baton-greenhouse@main

baton-greenhouse --username=API_TOKEN

baton resources
```

# Data Model

`baton-greenhouse` will pull down information about the following resources:
  - Users
  - Roles for Job Permissions
  - Roles for Future Job Permissions

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually
building spreadsheets. We welcome contributions, and ideas, no matter how
small&mdash;our goal is to make identity and permissions sprawl less painful for
everyone. If you have questions, problems, or ideas: Please open a GitHub Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-greenhouse` Command Line Usage

```
baton-greenhouse

Usage:
  baton-greenhouse [flags]
  baton-greenhouse [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  config             Get the connector config schema
  help               Help about any command

Flags:
      --client-id string                                 The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string                             The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
      --external-resource-c1z string                     The path to the c1z file to sync external baton resources with ($BATON_EXTERNAL_RESOURCE_C1Z)
      --external-resource-entitlement-id-filter string   The entitlement that external users, groups must have access to sync external baton resources ($BATON_EXTERNAL_RESOURCE_ENTITLEMENT_ID_FILTER)
  -f, --file string                                      The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                                             help for baton-greenhouse
      --log-format string                                The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string                                 The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
      --on_behalf_of_email string                        Email of the Site Admin user ($BATON_ON_BEHALF_OF_EMAIL)
      --otel-collector-endpoint string                   The endpoint of the OpenTelemetry collector to send observability data to (used for both tracing and logging if specific endpoints are not provided) ($BATON_OTEL_COLLECTOR_ENDPOINT)
  -p, --provisioning                                     This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-full-sync                                   This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --ticketing                                        This must be set to enable ticketing support ($BATON_TICKETING)
      --username string                                  required: The username is your Greenhouse API token ($BATON_USERNAME)
  -v, --version                                          version for baton-greenhouse

Use "baton-greenhouse [command] --help" for more information about a command.
```
