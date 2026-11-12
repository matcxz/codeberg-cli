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

For browser-based login, register a **public OAuth application** in the
instance's application settings, set its callback URL to
`http://127.0.0.1/`, then pass its client ID:

```sh
./cb auth login --web --client-id YOUR_CLIENT_ID
```

The flow uses OAuth authorization code with PKCE and a temporary loopback
callback. Use `--no-browser` to print the authorization URL when opening the
browser manually on the same machine. The browser must be able to reach the
machine running `cb` at
`127.0.0.1`; `--no-browser` only prints the URL and does not provide a remote
device handoff. Forgejo currently supports authorization-code OAuth, not the
OAuth Device Authorization Grant used by `gh auth login --web`, so `cb` cannot
show a one-time code for entering on another device. Browser login needs an
OAuth application client ID registered on each instance. See the [Forgejo
OAuth2 provider documentation](https://forgejo.org/docs/latest/user/authentication/oauth2-provider/).
Forgejo also does not implement OAuth permission scopes; approving an
application may grant broad account access. A scoped personal access token
remains preferable when least privilege is important.

For automation, set `CODEBERG_TOKEN`; it takes precedence over a saved token
and is never written to the config file. `CODEBERG_HOST` selects another
Forgejo instance, and `--host` overrides it for one invocation. HTTPS is
required except for loopback hosts.

For OAuth logins with a refresh token, `cb` refreshes the saved access token
automatically. Credentials are bound to their configured instance; changing
the host does not send a saved token to the other instance.

## Commands

```sh
cb repo list [--limit 30] [--page N] [--owner USER]
cb repo view OWNER/NAME
cb repo create NAME [--description TEXT] [--private] [--init]
cb issue list [--repo OWNER/NAME] [--state open|closed|all] [--page N]
cb issue view NUMBER [--repo OWNER/NAME]
cb issue create [--repo OWNER/NAME] --title TITLE [--body TEXT]
cb issue comment NUMBER [--repo OWNER/NAME] --body TEXT
cb issue close NUMBER [--repo OWNER/NAME]
cb issue reopen NUMBER [--repo OWNER/NAME]
cb pr list [--repo OWNER/NAME] [--state open|closed|all] [--page N]
cb pr view NUMBER [--repo OWNER/NAME]
cb pr create [--repo OWNER/NAME] --title TITLE --head BRANCH --base BRANCH
cb browse OWNER/NAME [LOCATION] [--no-browser]
```

Listing the authenticated user's repositories, creating repositories, and
mutating issues or pull requests requires authentication. Public resources can
be listed or viewed without a token; if a configured token lacks the scope for
a public read, `cb` retries the request without authentication. Accessing
private resources requires a token with the appropriate Forgejo API scopes.
Issue and pull request commands infer the repository from the current Git
`origin` remote when `--repo` is omitted; they accept `-R` as shorthand for
`--repo`. List commands support `--limit` and `--page`.
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
