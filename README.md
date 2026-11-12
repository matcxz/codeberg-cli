# Codeberg CLI

`cb` is a command-line client for Codeberg and other Forgejo instances.
It is published on GitHub and licensed under MIT.

## Requirements

- Go 1.27 or newer to build from source
- A Codeberg personal access token to use authenticated commands

## Build

```sh
go build -o cb .
```

## Authentication

Create a personal access token in Codeberg under **Settings → Applications**,
then log in. The token is requested without echo and stored in
`$XDG_CONFIG_HOME/cb/config.json` (or `~/.config/cb/config.json`) with
permissions restricted to the current user.

```sh
./cb auth login
./cb auth status
./cb auth logout
```

For automation, set `CODEBERG_TOKEN`; it takes precedence over a saved token
and is never written to the config file. `CODEBERG_HOST` selects another
Forgejo instance, and `--host` overrides it for one invocation. HTTPS is
required except for loopback hosts.

## Commands

```sh
cb repo list [--limit 30] [--owner USER]
cb repo view OWNER/NAME
cb repo create NAME [--description TEXT] [--private] [--init]
cb issue list --repo OWNER/NAME [--state open|closed|all]
cb issue view NUMBER --repo OWNER/NAME
cb issue create --repo OWNER/NAME --title TITLE [--body TEXT]
cb pr list --repo OWNER/NAME [--state open|closed|all]
cb pr view NUMBER --repo OWNER/NAME
cb pr create --repo OWNER/NAME --title TITLE --head BRANCH --base BRANCH
cb browse OWNER/NAME [LOCATION] [--no-browser]
```

Listing the authenticated user's repositories, creating repositories, and
creating issues or pull requests requires authentication. Public resources can
be listed or viewed without a token; if a configured token lacks the scope for
a public read, `cb` retries the request without authentication. Accessing
private resources requires a token with the appropriate Forgejo API scopes.
The issue and pull request commands accept `-R` as shorthand for `--repo`.
`browse` opens a repository or a path such as `issues/12`; use `--no-browser`
to print the URL instead.

Use `--json` on repository, issue, and pull request commands for
machine-readable output. Add `--help` to any command for its options.

## Development

```sh
go test ./...
```

The client uses Forgejo's `/api/v1` REST API. Forgejo documents token
authentication using the `Authorization: token ...` header and publishes the
instance's API reference at `/api/swagger` and OpenAPI document at
`/swagger.v1.json`.
