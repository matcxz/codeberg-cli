# Agent instructions

## Project

- `cb` is a Go CLI for Codeberg and other Forgejo instances.
- The Go module is `github.com/matcxz/codeberg-cli`; the built executable is `cb`.
- The MIT license applies. Do not publish this project to Codeberg; publish it on GitHub.

## Development

- Use Cobra commands in `cmd/` and keep Forgejo HTTP/API logic in `internal/forgejo/`.
- Store user configuration in `internal/config/`; never print or log access tokens.
- Keep saved credentials bound to their configured Forgejo host; do not send them to a `--host` override.
- OAuth login must use authorization code with PKCE, a loopback callback, random `state`, and HTTPS for remote instances. Forgejo does not support OAuth device authorization.
- Use the Forgejo `/api/v1` API. Do not require credentials for public reads; retry unauthenticated when a token is rejected only for missing scopes.
- Preserve HTTPS for remote hosts and the existing permission protections for saved credentials.
- Add focused tests for new behavior and mock HTTP APIs with `httptest`.
- Run `gofmt`, `go test ./...`, and `go vet ./...` before considering a change complete.
- Keep terminal output readable at narrow widths and provide `--json` for scripting where appropriate.
