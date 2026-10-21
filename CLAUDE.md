# okctl — Ownkube CLI

## Quick Reference

- **Binary**: `okctl`
- **Module**: `github.com/ownkube/okctl`
- **Build**: `make build` → `bin/okctl`
- **Test**: `make test` or `go test ./...`
- **Lint**: `make lint` (requires golangci-lint)
- **Regenerate API client**: `make generate` (requires oapi-codegen)

## Public repo: don't leak internals

This repo is **public** (github.com/ownkube/ownkube-cli). Everything committed is
world-readable: code, comments, strings, this file, and commit messages. Describe
what okctl does, never how the platform works behind the API:

- No names of internal services, infrastructure vendors, or the tooling the
  platform runs on (the app repo's banned-terms list applies here in full),
  and no private repo paths or design-doc numbers.
- No routing, scaling, sleep/wake, draining, or deploy mechanics of the server
  side, and no internal hostnames, namespaces, ports, or environment details.
- Comments justify client behaviour in client terms ("a hostname isn't accepted
  for every target"), not by explaining the server ("the server only routes X").

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

## Rebuild from updated spec

Where the updated spec comes from is in `CLAUDE.local.md` (gitignored; not in
public clones). With the new spec in hand:

```sh
make generate SPEC_SRC=<path to openapi.json>   # copy over api/openapi.json + regen
make build
```

`make generate` (with or without `SPEC_SRC`) runs `oapi-codegen` on
`api/openapi.json` to update `internal/api/client.gen.go`. After regen, add a wrapper to
`internal/client/client.go` for any new endpoint, then wire a cobra command
in the appropriate `cmd/<resource>/` subpackage.

Install `oapi-codegen` once if missing:

```sh
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0
```

Or run it without installing (after copying the spec):
`go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 -config oapi-codegen.yaml api/openapi.json`.

## Architecture

- **cmd/**: Cobra commands organised into subpackages — one per top-level
  resource. The root package only builds the root command and wires subtrees.
  - `cmd/root.go`: root command, flag/env/config resolution, `init()` hooks
    subpackages via `rootCmd.AddCommand(auth.Login(), ..., deploy.New(), ...)`.
  - `cmd/auth/`: `login`, `logout`, `whoami`. `SignIn(cmd)` runs the browser
    login for another command (`setup agent`).
  - `cmd/config/`: `config get|set|view` (CLI prefs) plus settings-as-code
    `pull|plan|apply [app]` against `ownkube.yaml`. `spec.go` holds the logic:
    `specFor` maps a deployment type to its PATCH config block, `toManifest`
    (secrets by name only, never values; sensitive keys and the keys in
    `heldBack` are never written), `overlay` (the PATCH replaces the whole
    block, so the yaml is laid over the live block and unmanaged keys are
    echoed back), `diff`/`diffVars`, and `makePlan` whose `hash()` is the plan
    id `apply --plan` checks. `plan --detect-changes` exits 2 on drift
    (`ux.ExitError`). Apply sends settings first (412 → "unlocks after the first
    build"), then variables via `UpdateVariables`.
  - `cmd/deploy/`: read verbs (`list|get|status|logs|revisions|connection`) plus
    the write/lifecycle surface (`create|update|delete|copy|move|promote|rollback|
    reset-password|job-runs|restart|rebuild|restore|maintenance|rename|
    auto-deploy|build-args|build-context|builder-size|subdomain`) and the
    read-only analytics reads (`observability|telemetry|cache-connection|
    build-logs`). `create`/`update` take a manifest via `-f` (`-` = stdin);
    `create` also has web convenience flags
    (`--name/--image/--tag/--port/--env/--public`). `delete` and `reset-password`
    confirm unless `--yes`. `subdomain` is a group (`set|check|suggest`).
    `logs` reads runtime logs (`--range`, or `--since/--until` parsed by
    `parseWhen` into the API's unix-second `start`/`end`, max 90 days; `-f`
    polls via `followLogs`, deduping on timestamp+message, NDJSON with `-o
    json`) or, with `--build [--latest|--revision]`, a revision's build log.
    `observability`/`telemetry` return nested metric blobs — they render as JSON
    even in table mode (`printBlob`). Client wrappers: write/lifecycle in
    `internal/client/deployment_write.go` + `deployment_actions.go` (reads stay in
    `client.go`); table/JSON render helpers in `cmd/deploy/render.go`. `move
    --project <id>` re-files a deployment (and its managed DB/cache group) under
    another project (control-plane-only; wrapper `MoveToProject` in
    `internal/client/functions.go`). Datastore verbs: `connection-info` (full
    credentials incl. password/URI), `link <app> <datastore> [--env-var]`
    (injects the connection string as a secret var + redeploys), and `resync`;
    wrappers in `internal/client/datastores.go`. Repo source: `source <id>`
    connects/switches an app's GitHub/GitLab repo (PUT `/source`; with no
    `--repo` the repo + branch come from this checkout's `origin` and current
    branch via `ux.ResolveRepoBranch`; confirms unless `--yes`; `--clear` →
    DELETE), `watch <id> [--revision]` follows the latest revision via
    `ux.FollowRevision`, and `rebuild --commit <sha>` builds one commit;
    `create --repo/--branch` sends the create body's `github` member. Commit
    helpers (`commitFlag`, `revisionCommit`; a repo build's revision `tag` is
    its commit) in `cmd/deploy/commit.go`. Wrappers in `internal/client/vcs.go`.
  - `cmd/environments/`: read verbs (`list|get`) plus the write surface
    (`create|update|set-env|delete`). `set-env` REPLACES the full shared env-var
    set (`--env`/`--secret KEY=VALUE`, repeatable, or `-f` JSON array) and
    redeploys the environment's apps; `delete` confirms unless `--yes`. Client
    wrappers live in `internal/client/environment_write.go` (reads in `client.go`);
    color validation + render helpers in `cmd/environments/helpers.go`.
  - `cmd/projects/`: `projects list|get|create|update|delete` — the top-level
    workspace grouping (Org → Project → Environment → App). Project IDs feed
    `--project` on `deploy move` and (server-side) the `projectId` deployment/
    environment filters. `delete` confirms unless `--yes` and is gated server-side
    (never the Default, never with a live deployment). Wrappers in
    `internal/client/projects.go`; color validation + render in
    `cmd/projects/helpers.go`.
  - `cmd/functions/` (alias `fn`): `functions list|source|deploy` — cluster-less
    Compute (ember) functions. `source` prints the inline file (`--code-only` for
    just the code); `deploy` takes a combined code+config manifest via `-f` (`-` =
    stdin), one revision. Wrappers in `internal/client/functions.go`.
  - `cmd/aws/`: `aws connect|list|get|verify|reconnect|resync|delete`. `connect`
    handles browser handoff (default), autonomous `--deploy` (shells out to the
    `aws` CLI), and polls the account until `verified`/`failed`.
  - `cmd/regions/`: `regions list` — Ownkube Compute regions you can deploy to.
    Wrappers in `internal/client/regions.go`.
  - `cmd/boxes/`: `boxes list [--kind web|database|cache]` — the Compute box
    catalog; a box id is the `skuId` for a Compute database/cache. Wrapper
    `ListBoxes` in `internal/client/datastores.go`.
  - `cmd/marketplace/` (alias `mp`): `list|get <slug>|check <slug>|deploy
    <slug>|delete <install-id>` — ready-made apps deployed with their DB/cache
    in one call. `--input KEY=VALUE` values go up as strings (server coerces to
    the declared type); `deploy` takes `--region` or `--cluster`, plus
    `--db-box`/`--cache-box` on Compute. `delete` removes every piece of one
    install (`templateInstanceId`, also on `deploy get -o json`) and confirms
    unless `--yes`. Wrappers in `internal/client/marketplace.go`.
  - `cmd/usage/`: `usage current|month-to-date (mtd)|projected|history` — Compute
    usage and cost. Wrappers in `internal/client/usage.go`.
  - `cmd/billing/`: `billing wallet|credit (credit redeem <code>)|spend-controls
    (get|set)|subscribe|top-up|portal` — the prepaid wallet, credit, promo-code
    redemption, spend controls, and Polar checkout handoff (`--no-browser` to
    print the URL only). Wrappers in `internal/client/billing.go`.
  - `cmd/alerts/`: `alerts list|create|update|delete|firings` — per-deployment
    workload alert rules and firing history. Wrappers in
    `internal/client/alerts.go`.
  - `cmd/domains/` (alias `custom-domains`): `domains list|add|verify|delete
    (unlink)` — link custom hostnames to a deployment. Wrappers in
    `internal/client/domains.go`.
  - `cmd/link/`: top-level `link` / `unlink` (NOT a `New()` group — exports
    `Link()` + `Unlink()`, registered directly). Binds the current directory to
    an org/cluster/environment/deployment for later commands to infer their
    target; `--init` scaffolds an `ownkube.yaml`. Interactive pickers via
    `prompt.Select`, or `--organization/--cluster/--environment/--service` for
    CI. Structured mode errors instead of prompting. No client wrapper — reuses
    existing list/get reads. Imports `internal/link` aliased as `linkstore`.
  - `cmd/up/`: top-level `up` (exports `New()`). Builds + deploys the current
    directory's LOCAL working tree (uncommitted edits included), not a committed
    git revision. Flow: presign an upload slot → tar+gzip the tree (`archive.go`;
    honours `.gitignore` via `git ls-files -co --exclude-standard`, else walks
    minus `.git`) → PUT to the pre-signed URL → trigger build → follow streams
    build logs + revision status to a terminal state (`ux.FollowRevision`). Target deployment from
    `--service` or this directory's link; `--note`, `--path`, `--no-follow`.
    Unlinked + no `--service` → `create.go`: presign with no deployment (server
    refuses an empty wallet before upload), POST `/v1/source-deploys` with
    `nameHint` = link-root dir name, print the server's `defaults`, write the
    `links.yaml` binding. Defaults are resolved ONLY server-side; the create-only
    flags (`--name/--region/--port/--type/--public/--project/--environment`) are
    sent only when set and rejected on a redeploy. Archive size checked against
    the presign's `maxBytes` before upload.
    Wrappers in `internal/client/source_upload.go` (`PresignSourceUpload`,
    `UploadSource` — bare `net/http` PUT, the URL self-authenticates —
    `DeployFromUpload`, `DeployUploadedSource`). `--repo/--branch` (`repo.go`)
    skip the upload and connect an existing (linked) app to the repo, same call
    as `deploy source`; refused when `up` would create the app. In a git
    checkout whose app isn't repo-built yet, `up` ends with a one-line tip
    pointing at `deploy source`.
    `--json`/`--ci` follow with the log on stderr and print one JSON result
    (`outcome.go`), non-zero exit on failure.
  - `cmd/ssh/` (alias `shell`): `ssh [app] [--instance N] [-- cmd]` (shell / one-shot command on
    a Compute app over real SSH), `ssh keys list|add|remove|github`, `ssh config`
    (managed `# okctl:begin/end <alias>` blocks in `~/.ssh/config`). Package is
    imported as `sshcmd` in root.
  - `cmd/connect/`: `connect [id] [--tunnel-only] [--port] [--ssh]` — psql /
    valkey-cli against a Compute database/cache: public URI when ready, else a
    local SSH tunnel.
  - `cmd/vcs/`: `vcs connections [--open]|repos [--connection]|branches
    <owner/repo> [--connection]` — read-only listings of the org's connected
    GitHub/GitLab accounts. Connecting an account is dashboard-only;
    `connections --open` opens `connectUrl`. Wrappers in
    `internal/client/vcs.go`.
  - `cmd/status/`: top-level `status` (alias `overview`) — the linked (or
    `-p`) project's apps, environments, URLs and health.
  - `cmd/vars/` (aliases `variables`, `var`): `vars list [app] [--kv]|set [app]
    K=V... [--secret] [--stdin]|delete [app] K... [-y]|edit [app]`. Secret
    values are write-only (never read back; `edit` shows `<secret>` and refuses
    edits to them); environment-shared vars are listed, never unset. `set`'s
    custom Args: an arg containing `=` can't be the app. Wrappers in
    `internal/client/variables.go` (`ListVariables`, `UpdateVariables`,
    `ResolveVariables`).
  - `cmd/run/`: top-level `run [app] [-- cmd...]` — resolves the app's
    variables (`purpose=run`, or `shell` with no command → `$SHELL`) and runs
    the command locally with them in its env only: never printed, written to
    disk, or put in argv. Exits with the child's status (`ux.ExitError`). Own
    `-s/-e/-p` flags.
  - `cmd/mcp/`: `mcp serve` — stdio ↔ the MCP endpoint (`<api-url>/api/mcp`,
    `client.MCP`): one JSON-RPC message per line, SSE `data:` events relayed
    one per line, `Mcp-Session-Id` echoed, an expired session re-initialized by
    replaying the agent's `initialize`; stdout carries only JSON-RPC (it's in
    root's `quietCommands`). `mcp install [--agent] [--print]` writes an
    `ownkube` server entry running `okctl mcp serve` (the API key stays in
    okctl's credential store, never in the agent's config); `Install` is
    reused by `setup agent`.
  - `cmd/skills/`: `skills install [--agent]|show` — writes the embedded
    `skills/okctl/SKILL.md` (package `skills` at the repo root, `go:embed`) into
    each agent's user skills dir.
  - `cmd/setup/`: `setup agent [--agent] [--no-mcp] [--no-skills]` — login if
    needed, then `mcp.Install` + `skills.Install`.
  - `cmd/apicall/`: `api <method> <path> [-d]` — raw authenticated request
    (`client.Do`); refuses full URLs so the key never leaves the API host.
  - `cmd/open/`, `cmd/docs/`: open an app in the dashboard / the docs
    (`https://ownkube.io/docs`); `--print` for the URL only.
  - `cmd/upgrade/`: `upgrade [--check]` — replaces a hand-installed binary from
    the latest GitHub release (checksum-verified, `internal/update`), or prints
    the package manager's upgrade command.
  - `cmd/internal/ux/`: shared helpers — `RequireClient`, `ResolveDeployment`
    (flag, else this directory's link), `Print`, `Deref`,
    `UnionString` (renders generated `oneOf` scalar unions, e.g. timestamps),
    `ReadFileOrStdin`, `IsStructured`, `APIURL`, `Config`, `OpenBrowser`,
    `FollowRevision`/`LatestRevision` (build-log + status follow loop shared by
    `up`, `deploy watch`), `ResolveRepoBranch` (flags, else git origin),
    name-or-id resolution (`resolve.go`: `ResolveApp`/`ResolveProject`/
    `ResolveEnvironment`, `ResolveScope`, `LooksLikeID`; an ambiguous name
    errors listing the ids), `WireAppTargets` (`target.go`, see "The `[app]`
    argument" below), and `ExitError` (`exit.go`: root exits with its `Code`,
    silently when `Err` is nil). Set once per invocation by `cmd/root.go`'s
    `PersistentPreRunE` via `ux.Set(...)`.
  - `completion.go`, `version.go`: small leaf commands kept flat.
- **internal/api/**: GENERATED Go client from `api/openapi.json` — do NOT edit.
- **internal/client/**: Thin wrapper around generated client (API key
  injection, optional HTTP basic auth via `OKCTL_BASIC_AUTH`, normalised
  error handling). One method per endpoint. Exception: `ssh.go` hand-writes
  the SSH endpoints through `raw.go`'s `doJSON` (returns `*APIError`,
  test with `IsAPICode`) until `api/openapi.json` ships them; then regenerate
  and switch to generated calls. `checkError` also returns `*APIError`, so
  `IsAPICode` works for every endpoint. `mcp.go` (`MCP`, `MCPEndpoint`) is the
  one non-`/api/cli` call.
- **internal/agents/**: AI-agent setup shared by `mcp install`, `skills
  install` and `setup agent`: detection (`Select`; CLI on PATH or config dir,
  honouring `CLAUDE_CONFIG_DIR`/`CODEX_HOME`), `InstallMCP` (Claude Code via
  `claude mcp add --scope user`, Cursor `~/.cursor/mcp.json` merge, Codex
  `config.toml` table replace; other entries untouched, atomic writes),
  `InstallSkill`, and `OkctlServer` (absolute okctl path, since a desktop-
  launched agent may not share the shell's PATH).
- **internal/update/**: release lookup, the passive daily update notice, and
  self-replace for `upgrade`. Release archives are named after the repo:
  `ownkube-cli_<os>_<arch>.tar.gz` (`.zip` on Windows) + `checksums.txt`.
- **internal/sshconn/**: okctl's SSH client (`golang.org/x/crypto/ssh`, no
  OpenSSH needed). Key discovery/creation (`~/.ssh/ownkube_ed25519`, agent
  keys), `EnsureKey` first-connect flow, host keys pinned from the API (never
  trust-on-first-use) and mirrored into `~/.ssh/known_hosts`, `Shell`/`Run`
  (exit code propagated), `Forward` (redials a dropped session via
  `OpenRedialable` and keeps the local port), `~/.ssh/config` blocks. The SSH
  username is the deployment ID (`LoginUser`); a hostname isn't accepted for
  every target, so don't switch back to the API's `username`.
- **internal/gitrepo/**: Reads a checkout's `origin` (scp-like, ssh://,
  https:// forms → host + `owner/repo`, GitLab subgroups kept whole) and current
  branch (`Detect`, `ParseRemoteURL`); `SplitRepository` splits at the LAST
  slash, matching how the API reads GitLab `group/sub/project`.
- **internal/config/**: Config + credentials YAML file management
  (`~/.config/ownkube/`).
- **internal/link/**: Per-directory resource binding store — `links.yaml` in the
  config dir, keyed by git-root abs path (via `ResolveKey`), ids only, 0600,
  never committed. `Manager` with `Get/Set/Remove`.
- **internal/manifest/**: `ownkube.yaml` read/write — the committed, hand-editable
  deploy spec (`Load/Write/Exists/JSON`). No ids, no secret values. Feeds `deploy
  create -f` in later phases.
- **internal/output/**: Table/JSON/YAML output formatter.
- **internal/prompt/**: Terminal input helpers (secret input, confirmations,
  `Select` numbered picker). Prompts go to stderr so stdout stays clean.
- **internal/version/**: Build version info (set via ldflags).
- **api/openapi.json**: the CLI API's OpenAPI spec (source of truth for types).

## Conventions

- API types are generated from the OpenAPI spec — never hand-write them.
- Regenerate the client after spec changes: `make generate` (see "Rebuild" above).
- Config dir: `~/.config/ownkube/` with `config.yaml` (prefs) and
  `credentials.yaml` (0600, auth).
- API URL priority: `--api-url` flag > `OKCTL_API_URL` env > config.yaml >
  `https://app.ownkube.io`.
- `OKCTL_BASIC_AUTH=user:pass` adds HTTP Basic auth on every request, for an
  API URL served behind it.
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
5. If it acts on one app, make `[app]` the first positional in `Use` (see
   below) instead of taking a raw deployment id.

### The `[app]` argument

`ux.WireAppTargets(rootCmd)` (called once in `cmd/root.go` after every
`AddCommand`) wires each command whose `Use` has `[app]` as its first
positional and that has no `--service` flag of its own. It adds
`-s/--service`, `-e/--environment` and `-p/--project`, and appends the shared
help text. The command's `Args` validator accepts the args with or without the
app. Before `RunE` runs, the app (an argument, `--service`, or this directory's
link) is resolved by name or id to a deployment id and passed in as `args[0]`.
So write `RunE` as if `args[0]` were always an id, and declare `Args` for the
full form (e.g. `cobra.ExactArgs(2)` for `rollback [app] <revision-id>`).
Variadic commands need a custom validator that can tell the app from the other
args (`vars set`: an arg containing `=` is never the app). A command with its
own `--service` (`ssh`, `connect`, `run`) is skipped and resolves itself with
`ux.ResolveScope` + `ux.ResolveApp`.

## Adding a New API Endpoint

1. Get the endpoint into the API spec (see `CLAUDE.local.md`).
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
