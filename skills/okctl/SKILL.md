---
name: okctl
description: Use this skill when the user wants to deploy or operate apps on Ownkube with the okctl CLI. That covers web apps, background workers, scheduled jobs, Postgres databases, Valkey caches and functions on Ownkube Compute (no cloud account needed); deploying the local working directory; reading logs and status; rolling back; wiring an app to a database or cache; custom domains and alerts; and checking usage or the prepaid wallet. Also use it when the user mentions Ownkube or okctl by name.
---

# okctl, the Ownkube CLI

`okctl` is the CLI for [Ownkube](https://ownkube.io). Ownkube Compute runs your web apps, workers, jobs, Postgres, Valkey caches and functions with no cloud account and no cluster to manage. You pay from a prepaid wallet. The same app can later run in your own AWS account (GCP and Azure coming soon) with no rewrite, keeping the same deploys and observability.

## Install

```bash
brew install ownkube/tap/okctl
# or
go install github.com/ownkube/okctl@latest
```

A command in this skill missing from `okctl --help`? The binary is older than the skill: `brew upgrade okctl`.

## Auth

```bash
okctl login      # browser sign-in, writes ~/.config/ownkube/credentials.yaml
okctl status     # who is signed in, which org
okctl logout
```

Production (`https://app.ownkube.io`) needs only the key from `okctl login`. `OKCTL_BASIC_AUTH=user:pass` is only for an API URL served behind HTTP Basic auth. Never set it for `https://app.ownkube.io`.

## okctl or the MCP server

If the agent is already connected to the Ownkube MCP server (`https://app.ownkube.io/api/mcp`), use its tools for most operations. Reach for okctl when:

- deploying the **local working directory** (`okctl up` uploads it, honouring `.gitignore`)
- the user needs **live credentials** (`deploy connection-info`, `deploy reset-password`) or a **point-in-time restore**: run these only when the user asks, and show the output to them rather than copying secrets into files or chat
- scripting in CI, or no MCP connection is set up

## Output: use `-o json` for parsing

Never parse table output. Every command supports `-o json` and `-o yaml`.

```bash
okctl deploy list --environment env_abc -o json | jq '.[] | select(.status == "failed") | .id'
```

## Workflows

**Deploy this directory (first time):**
```bash
okctl regions list                      # pick a region id
okctl up --name my-app --region <region-id> --port 3000
```
In an unlinked directory, `okctl up` creates a new Compute web app, links the directory to it and follows the build until it is live. Use `--type worker` for a worker, `--public=false` to keep it private, `--project` / `--environment` to place it.

**Redeploy:**
```bash
okctl up --note "fix login redirect"    # linked directory: uploads and follows the build
okctl up --no-follow                    # return as soon as the build starts
```
To target an existing app first: `okctl link` (interactive) or `okctl link --organization <id> --environment <id> --service <deployment-id>` (non-interactive, e.g. CI). The binding lives in your global config, never in the repo. `okctl link --init` also scaffolds an `ownkube.yaml` deploy spec.

**Add a Postgres database and wire it in:**
```bash
okctl boxes list --kind database -o json    # box id = skuId
cat > db.yaml <<'EOF'
resourceType: database
name: my-db
region: <region-id>
config:
  engine: postgres
  version: "18"
  skuId: <box-id>
  billingMode: reserved
  bootstrap: { database: app, owner: app }
  storage: { size: 10Gi }
EOF
okctl deploy create -f db.yaml
okctl deploy link <app-id> <db-id>          # sets DATABASE_URL as a secret variable and redeploys
```
A Valkey cache is the same shape with `resourceType: cache`, `config.engine: valkey`, `config.version: "8.1"`, a box from `okctl boxes list --kind cache`, and no `bootstrap` / `storage`. `deploy link` sets `REDIS_URL` for a cache. Override the name with `--env-var`.

**Deploy a ready-made app (n8n, Directus, ...) with its database and cache wired in:**
```bash
okctl marketplace list
okctl marketplace get n8n -o json | jq '{components, inputs}'   # what it creates, inputs to collect
okctl marketplace check n8n --input KEY=VALUE                   # dry run, creates nothing
okctl marketplace deploy n8n --region <region-id> --db-box <box-id> [--cache-box <box-id>] [--input KEY=VALUE]
okctl marketplace delete <install-id>                           # removes every piece, data included
```
`deploy` prints the install id and every deployment it created; follow each with `okctl deploy status <id>`.

**Diagnose a failing app:**
```bash
DEP=d_abc
okctl deploy status $DEP -o json | jq '{status, sync, health, statusMessage}'
okctl deploy logs $DEP --range 600 --limit 500 --filter error
okctl deploy revisions $DEP -o json | jq '.[0:3] | .[] | {id, status, failureReason}'
okctl deploy build-logs $DEP <revision-id>   # when the build itself failed
okctl deploy rollback $DEP <revision-id> --note "bad release"
okctl deploy resync $DEP                     # re-apply settings to recover a stalled rollout
```

**Money:**
```bash
okctl billing wallet          # balance
okctl usage mtd               # month to date
okctl usage projected         # projected month cost
okctl billing top-up 20       # USD; opens checkout
```

## Command reference

```bash
# Deploy from a directory
okctl up [--service <dep-id>] [--path <dir>] [--note <text>] [--no-follow]
         [--name --region --port --type web|worker --public --project --environment]   # new app only
okctl link [--organization --cluster --environment --service] [--init]
okctl unlink [-y]

# Deployments (web, worker, job, database, cache, function)
okctl deploy list --environment <id> | --cluster <id>     # exactly one
okctl deploy get | status | revisions | observability | telemetry <dep-id>
okctl deploy logs <dep-id> [--range <seconds>] [--limit N] [--filter <substring>]
okctl deploy build-logs <dep-id> <revision-id>
okctl deploy create -f <file|->                           # JSON or YAML, top-level resourceType
okctl deploy create --name <n> --image <img> --tag <t> --port <p> [--public] [--env KEY=VALUE]...
okctl deploy update <dep-id> -f <file|->
okctl deploy rollback <dep-id> <revision-id> [--note <text>]
okctl deploy restart | rebuild | resync <dep-id>
okctl deploy rename | copy | promote <dep-id> ...
okctl deploy move <dep-id> --project <id>
okctl deploy maintenance | auto-deploy <dep-id> ...
okctl deploy build-args | build-context <dep-id> ...
okctl deploy builder-size <dep-id> <small|standard|large>
okctl deploy subdomain set | check | suggest ...
okctl deploy job-runs list | trigger | cancel ...
okctl deploy delete <dep-id> [-y]

# Databases and caches
okctl deploy link <app-id> <datastore-id> [--env-var NAME]
okctl deploy connection <dep-id>                           # connection secret name and details
okctl deploy connection-info <datastore-id>                # LIVE credentials
okctl deploy cache-connection <cache-id>
okctl deploy reset-password <db-id>
okctl deploy restore <db-id> ...                           # point-in-time recovery

# Functions (alias: fn)
okctl functions list
okctl functions source <dep-id> [--code-only]
okctl functions deploy <dep-id> -f <file>

# Organize
okctl organizations list                                   # aliases: orgs, org
okctl projects list | get | create | update | delete
okctl environments list | get | create | update | delete   # aliases: envs, env
okctl environments set-env <env-id> --env K=V --secret K=V | -f <file>   # REPLACES the full set
okctl regions list
okctl boxes list [--kind web|database|cache]
okctl marketplace list | get <slug> | check <slug> | deploy <slug> | delete <install-id>   # alias: mp
okctl registries list | get <id>

# Domains and alerts
okctl domains list <dep-id> | add <dep-id> <hostname> | verify <domain-id> | delete <domain-id>
okctl alerts list <dep-id> | create <dep-id> | update <rule-id> | delete <rule-id> | firings

# Usage and billing
okctl usage current | mtd | projected | history
okctl billing wallet | top-up <usd> | portal [--no-browser]
okctl billing credit | credit redeem <code>
okctl billing spend-controls get | set
okctl billing subscribe <personal|team>

# Your own cloud (optional)
okctl aws connect [--deploy] | list | get | verify | reconnect | resync | delete
okctl clusters list | get | create | status | cancel | delete

# CLI
okctl config get | set | view
okctl completion <bash|zsh|fish|powershell>
okctl version
```

Run `okctl <command> --help` for the exact flags of anything listed with `...`.

## Config precedence

`flag > env var > ~/.config/ownkube/config.yaml > default`

| Setting | Flag | Env var | Default |
|---|---|---|---|
| API URL | `--api-url` | `OKCTL_API_URL` | `https://app.ownkube.io` |
| Output format | `-o, --output` | none | `table` |
| Basic Auth | none | `OKCTL_BASIC_AUTH` (`user:pass`) | none |

## Errors and fixes

| Error | Fix |
|---|---|
| `not logged in ... run 'okctl login' first` | `okctl login` |
| `API error 401: Missing username and password` | The API URL is behind HTTP Basic auth: set `OKCTL_BASIC_AUTH=user:pass`. Never needed for `https://app.ownkube.io`. |
| `API error 404: 404 Not Found` | Wrong API URL: check `okctl config view` |
| `specify exactly one of --cluster or --environment` | Pass one (not both, not neither) to `okctl deploy list` |
| `unknown command` | The binary is older than this skill: `brew upgrade okctl` |

## Rules

- **Confirm with the user before anything destructive or costly**: `delete`, `reset-password`, `restore`, `rollback`, `top-up`, `subscribe`, `clusters create`. Only pass `-y` / `--yes` once they have said yes.
- `environments set-env` **replaces** every variable in the environment. Read the current set first (`environments get -o json`) and send the full list.
- Prefer `deploy link` over copying `connection-info` output into env vars, files or chat.
- Don't read or edit `~/.config/ownkube/credentials.yaml`; use `okctl login` / `logout`.
- Don't parse table output; use `-o json`.
- Don't `curl` the API unless explicitly asked; the CLI handles auth and errors.

## Links

[Docs](https://ownkube.io/docs) · [Repo](https://github.com/ownkube/ownkube-cli) · [Releases](https://github.com/ownkube/ownkube-cli/releases) · [Tap](https://github.com/ownkube/homebrew-tap)
