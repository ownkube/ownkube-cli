# okctl — Ownkube CLI

## Quick Reference

- **Binary**: `okctl`
- **Module**: `github.com/ownkube/okctl`
- **Build**: `make build` → `bin/okctl`
- **Test**: `make test` or `go test ./...`
- **Lint**: `make lint` (requires golangci-lint)
- **Regenerate API client**: `make generate` (requires oapi-codegen)

## Subagent model selection (core instruction)

When spawning subagents via the Agent/Task tool for work in this repo, **always pass `model` explicitly.** `sonnet` is the default; `opus` is the exception you have to justify; `haiku` is the floor for mechanical, check-guarded fan-out. Judge the task, not the tool it happens to use. "The agent writes Go" is not on its own a reason to reach for opus.

Spend `opus` when at least one of these is true:
- **The shape isn't decided yet.** Designing a new command tree, a client abstraction, an auth/token flow, or a config format, anything where a plausible-looking wrong answer costs more than the model does.
- **Diagnosis is the work.** Debugging a flaky test, a codegen mismatch after `make generate`, or a failure that only reproduces against a live API.
- **The blast radius is wide.** Auth and credential handling, the API client contract, release/build tooling, anything destructive or hard to reverse.
- **It needs taste.** User-facing CLI help text, command names, and output formatting where "correct" and "actually good" come apart.

Use `sonnet` for everything else, including plenty of work that writes files: exploration and search ("where is command X wired"), mechanical edits against a decided spec (adding a wrapper for a new endpoint, wiring a cobra command that follows an existing `cmd/<resource>/` pattern, applying a rename across call sites), and fan-out where every worker gets the same solved example.

Drop to `haiku` for the mechanical floor of that fan-out — a decided edit applied verbatim where a machine check (`go build`/`go vet`/`go test`) catches any slip and the worker never decides *how*, only transcribes: a rename across call sites, a version bump across files, wiring the Nth endpoint wrapper from an already-solved example. The test: if you could nearly write a codemod for it, it's haiku work. Step up to `sonnet` the moment the worker must read surrounding code to decide how to apply the change, preserve non-obvious behavior, or exercise any taste.

Rule of thumb: ask what the agent has to *decide*. If the decisions are made and it is executing them, sonnet, and give it the worked example (an existing command or wrapper) that makes that true; drop to haiku when the edit is purely mechanical and a check will catch any slip. If it makes a call you would want to review, opus. For a mixed batch, solve the hard instance yourself first, then fan out on sonnet (or haiku).

This mirrors the ecosystem rule in `../CLAUDE.md` and is repeated here as a core instruction.

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
  - `cmd/deploy/`: read verbs (`list|get|status|logs|revisions|connection`) plus
    the write/lifecycle surface (`create|update|delete|copy|promote|rollback|
    reset-password|job-runs|restart|rebuild|restore|maintenance|rename|
    auto-deploy|build-args|build-context|builder-size|subdomain`) and the
    read-only analytics reads (`observability|telemetry|cache-connection|
    build-logs`). `create`/`update` take a manifest via `-f` (`-` = stdin);
    `create` also has web convenience flags
    (`--name/--image/--tag/--port/--env/--public`). `delete` and `reset-password`
    confirm unless `--yes`. `subdomain` is a group (`set|check|suggest`).
    `observability`/`telemetry` return nested metric blobs — they render as JSON
    even in table mode (`printBlob`). Client wrappers: write/lifecycle in
    `internal/client/deployment_write.go` + `deployment_actions.go` (reads stay in
    `client.go`); table/JSON render helpers in `cmd/deploy/render.go`.
  - `cmd/environments/`: read verbs (`list|get`) plus the write surface
    (`create|update|set-env|delete`). `set-env` REPLACES the full shared env-var
    set (`--env`/`--secret KEY=VALUE`, repeatable, or `-f` JSON array) and
    redeploys the environment's apps; `delete` confirms unless `--yes`. Client
    wrappers live in `internal/client/environment_write.go` (reads in `client.go`);
    color validation + render helpers in `cmd/environments/helpers.go`.
  - `cmd/aws/`: `aws connect|list|get|verify|reconnect|resync|delete`. `connect`
    handles browser handoff (default), autonomous `--deploy` (shells out to the
    `aws` CLI), and polls the account until `verified`/`failed`.
  - `cmd/regions/`: `regions list` — Ownkube Compute regions you can deploy to.
    Wrappers in `internal/client/regions.go`.
  - `cmd/usage/`: `usage current|month-to-date (mtd)|projected|history` — Compute
    usage and cost. Wrappers in `internal/client/usage.go`.
  - `cmd/billing/`: `billing wallet|credit (credit claim)|spend-controls (get|set)|
    subscribe|top-up|portal` — the prepaid wallet, credit, spend controls, and
    Polar checkout handoff (`--no-browser` to print the URL only). Wrappers in
    `internal/client/billing.go`.
  - `cmd/alerts/`: `alerts list|create|update|delete|firings` — per-deployment
    workload alert rules and firing history. Wrappers in
    `internal/client/alerts.go`.
  - `cmd/domains/` (alias `custom-domains`): `domains list|add|verify|delete
    (unlink)` — link custom hostnames to a deployment. Wrappers in
    `internal/client/domains.go`.
  - `cmd/internal/ux/`: shared helpers — `RequireClient`, `Print`, `Deref`,
    `ReadFileOrStdin`, `IsStructured`, `APIURL`, `Config`, `OpenBrowser`. Set
    once per invocation by `cmd/root.go`'s `PersistentPreRunE` via `ux.Set(...)`.
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
3. Add a method to `internal/client/client.go` (or a topic file like
   `deployment_write.go`) wrapping the generated call and reusing
   `checkError(...)` + a per-endpoint `errorsFromXxx` adapter.
4. Wire a cobra command in the relevant `cmd/<resource>/` subpackage (see
   "Organising new code").

**Gotcha — required+nullable fields are VALUE types (oapi-codegen v2.8.0 /
OpenAPI 3.1).** A field that is both required and nullable renders as a value
type (`string`, a value struct, `map[string]interface{}`) — NOT a pointer.
Only *optional* fields (with `omitempty`) render as `*T`. So call `ux.Deref`
and `!= nil` guards on optional pointer fields only; read required-nullable
fields directly (null/absent JSON unmarshals to the zero value, which prints
the same as `Deref` did). When in doubt, grep the generated struct in
`internal/api/client.gen.go` — the `json:"...,omitempty"` tag marks the
pointers. (v2.4.1 rendered required-nullable as `*T`; the v2.8.0 upgrade
flipped this and broke every deref call site.)

**Gotcha — `oneOf` request bodies.** oapi-codegen renders an OpenAPI `oneOf`
body (e.g. the deployment `create` union) as a struct with an *unexported*
`union json.RawMessage` field and no marshal helper, so the typed
`...WithResponse(ctx, body)` call sends `{}`. For such endpoints send raw bytes
via the `...WithBodyWithResponse(ctx, "application/json", bytes.NewReader(b))`
variant instead (see `CreateDeployment`/`UpdateDeployment`). Manifests are read
with `ux.ReadFileOrStdin` and normalised through `manifestToJSON` (YAML⊃JSON).
