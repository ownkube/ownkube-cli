# okctl — Ownkube CLI

## Quick Reference

- **Binary**: `okctl`
- **Module**: `github.com/ownkube/okctl`
- **Build**: `make build` → `bin/okctl`
- **Test**: `make test` or `go test ./...`
- **Lint**: `make lint` (requires golangci-lint)
- **Regenerate API client**: `make generate` (requires oapi-codegen)

## Rebuild from updated spec

Whenever the CLI API changes in `ownkube-app`, refresh the Go client and rebuild:

```sh
# 1. Regenerate the JSON spec in ownkube-app
( cd ../ownkube-app && pnpm cli:spec )

# 2. Copy the spec into api/openapi.json and regenerate the Go client + build
make generate
make build
```

`make generate` copies `../ownkube-app/src/services/cli/openapi.json` →
`api/openapi.json` and runs `oapi-codegen` to update
`internal/api/client.gen.go`. After regen, add a wrapper to
`internal/client/client.go` for any new endpoint, then wire a cobra command
in the appropriate `cmd/<resource>/` subpackage.

Install `oapi-codegen` once if missing:

```sh
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

## Architecture

- **cmd/**: Cobra commands organised into subpackages — one per top-level
  resource. The root package only builds the root command and wires subtrees.
  - `cmd/root.go`: root command, flag/env/config resolution, `init()` hooks
    subpackages via `rootCmd.AddCommand(auth.Login(), ..., deploy.New(), ...)`.
  - `cmd/auth/`: `login`, `logout`, `status`.
  - `cmd/config/`: `config get|set|view`.
  - `cmd/deploy/`: `deploy list|get|create|update|delete|status|logs|revisions|rollback|auto-deploy|connection`.
  - `cmd/internal/ux/`: shared helpers — `RequireClient`, `Print`, `Deref`,
    `ReadFileOrStdin`, `IsStructured`, `APIURL`, `Config`. Set once per
    invocation by `cmd/root.go`'s `PersistentPreRunE` via `ux.Set(...)`.
  - `completion.go`, `version.go`: small leaf commands kept flat.
- **internal/api/**: GENERATED Go client from `api/openapi.json` — do NOT edit.
- **internal/client/**: Thin wrapper around generated client (API key
  injection, optional HTTP basic auth via `OKCTL_BASIC_AUTH`, normalised
  error handling). One method per endpoint.
- **internal/config/**: Config + credentials YAML file management
  (`~/.config/ownkube/`).
- **internal/output/**: Table/JSON/YAML output formatter.
- **internal/prompt/**: Terminal input helpers (secret input, confirmations).
- **internal/version/**: Build version info (set via ldflags).
- **api/openapi.json**: OpenAPI spec copied from ownkube-app (source of truth).

## Conventions

- API types are generated from the OpenAPI spec — never hand-write them.
- Regenerate the client after spec changes: `make generate` (see "Rebuild" above).
- Config dir: `~/.config/ownkube/` with `config.yaml` (prefs) and
  `credentials.yaml` (0600, auth).
- API URL priority: `--api-url` flag > `OKCTL_API_URL` env > config.yaml >
  `https://api.ownkube.io`.
- `OKCTL_BASIC_AUTH=user:pass` adds HTTP Basic auth on every request — used
  for dev environments behind an HTTP gateway.
- Auth: browser-based flow — CLI opens browser to `/cli-authorize`, receives
  API key via local callback.
- Use `cmd.OutOrStdout()` for command output (testable).
- User-facing errors via `fmt.Errorf`, not `log.Fatal`.
- Output: branch on `ux.IsStructured()` — return `ux.Print(w, value)` for
  json/yaml, build a `[][]string` rows table otherwise.

## Organising new code (read before adding commands)

**Subpackage per resource.** Never add `cmd/<resource>_*.go` files. New
commands belong in `cmd/<resource>/` (create the directory if it does not
exist) with one constructor per command:

```go
// cmd/<resource>/<resource>.go
func New() *cobra.Command {
    root := &cobra.Command{Use: "<resource>", Short: "..."}
    root.AddCommand(listCmd(), getCmd(), ...)
    return root
}

// cmd/<resource>/list.go
func listCmd() *cobra.Command { ... }
```

Then register from `cmd/root.go`:

```go
rootCmd.AddCommand(resource.New())
```

**No reaching into the parent package.** Subpackages MUST NOT import `cmd`.
Use `cmd/internal/ux` for any shared state (API URL, output format, config
manager) and any helper that would otherwise be duplicated (`Deref`,
`ReadFileOrStdin`, `RequireClient`, `Print`, etc.). Add new helpers to ux
rather than copy-pasting between subpackages.

**Single Go file per command.** Each cobra command lives in its own file
within the subpackage (`list.go`, `get.go`, `rollback.go`, ...). The only
flat files in `cmd/` are the root wiring, completion, and version.

## Adding a New Command

1. If the resource subpackage doesn't exist yet, create `cmd/<resource>/`
   with a `New()` constructor and register it from `cmd/root.go`.
2. Add `cmd/<resource>/<verb>.go` exposing `func <verb>Cmd() *cobra.Command`.
3. Attach it inside `New()` via `root.AddCommand(<verb>Cmd())`.
4. If it needs auth or a client, call `ux.RequireClient()` at the start of
   `RunE` — do not hand-roll the auth check.

## Adding a New API Endpoint

1. Add the endpoint in `ownkube-app/src/services/cli/` (route file + register
   on `cliApp`).
2. Regenerate: see "Rebuild from updated spec" above.
3. Add a method to `internal/client/client.go` wrapping the generated call
   and reusing `checkError(...)` for error handling.
4. Wire a cobra command in the relevant `cmd/<resource>/` subpackage (see
   "Organising new code").
