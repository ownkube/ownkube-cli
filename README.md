# okctl

The command-line interface for the [Ownkube](https://app.ownkube.io) developer platform.

[![Release](https://img.shields.io/github/v/release/ownkube/ownkube-cli?sort=semver)](https://github.com/ownkube/ownkube-cli/releases/latest)
[![CI](https://github.com/ownkube/ownkube-cli/actions/workflows/ci.yaml/badge.svg)](https://github.com/ownkube/ownkube-cli/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ownkube/okctl.svg)](https://pkg.go.dev/github.com/ownkube/okctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

## Install

### Homebrew (macOS & Linux)

```sh
brew install ownkube/tap/okctl
```

The formula lives in [ownkube/homebrew-tap](https://github.com/ownkube/homebrew-tap).

### Pre-built binaries

Grab a tarball for your OS/arch from the [latest release](https://github.com/ownkube/ownkube-cli/releases/latest):

```sh
# macOS arm64 example
curl -sSL https://github.com/ownkube/ownkube-cli/releases/latest/download/ownkube-cli_darwin_arm64.tar.gz \
  | tar -xz -C /usr/local/bin okctl
```

Binaries are published for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, and `windows/arm64`.

### From source

```sh
go install github.com/ownkube/okctl@latest
```

## Quick start

```sh
# Log in via browser-based flow
okctl login
okctl whoami

# Build and deploy this directory (creates and links an app the first time)
okctl up

# The linked project's apps, environments, URLs and health
okctl status

# Follow an app's logs, set a variable, run a local command with its variables
okctl deploy logs api -f
okctl vars set api LOG_LEVEL=debug
okctl run api -- npm run dev

# Shell on the running app
okctl ssh api
```

Commands that act on one app take its name or id. In a directory linked with
`okctl link` (or deployed with `okctl up`), you can leave the app out. Use
`-e/--environment` or `-p/--project` when more than one app has the same name.

Use `--output json` (or `yaml`) on any read command for machine-readable output:

```sh
okctl deploy get api -o json | jq .
```

## Connect an AWS account

`okctl aws connect` onboards an AWS account end to end. Ownkube never receives
your AWS keys. The CloudFormation stack you deploy grants a scoped cross-account
role and phones home when it is ready.

```sh
# Handoff: opens the AWS console to a quick-create form, then waits for access
okctl aws connect

# Autonomous: deploy the stack with the AWS credentials already in your shell
# (requires the AWS CLI). Great for headless / agent use.
okctl aws connect --deploy

# Check status any time
okctl aws list
okctl aws get <account-id>
```

`connect` polls until the account is verified. Add `--no-wait` to return
immediately, `--no-browser` to print the URL without opening it, and `-o json`
to get the full payload (external ID + machine-readable stack template) for an
agent to drive the flow itself.

## Commands

| Group | What it does |
|---|---|
| `okctl login` / `logout` / `whoami` | Browser-based auth + credential management |
| `okctl status` | The linked project's apps, environments, URLs and health |
| `okctl up` | Build and deploy the current directory (`--json`/`--ci` for pipelines) |
| `okctl link` / `unlink` | Bind this directory to an app |
| `okctl deploy list\|get\|status\|watch\|logs\|revisions\|...` | Inspect and operate deployments |
| `okctl deploy source` | Connect an app to a repository so pushes deploy it |
| `okctl vars list\|set\|delete\|edit` | An app's variables (secret values are write-only) |
| `okctl run [app] [-- cmd]` | Run a local command, or your shell, with an app's variables |
| `okctl ssh` (alias `shell`) / `okctl connect` | Shell on a running app / `psql` or `valkey-cli` to a database or cache |
| `okctl config pull\|plan\|apply` | An app's settings as code in `ownkube.yaml` |
| `okctl config get\|set\|view` | Manage `~/.config/ownkube/config.yaml` |
| `okctl projects` / `environments` / `organizations` | Organize apps |
| `okctl domains` / `alerts` | Custom domains and alert rules |
| `okctl marketplace` | Ready-made apps deployed with their database and cache |
| `okctl usage` / `billing` | Usage, cost and the prepaid wallet |
| `okctl vcs connections\|repos\|branches` | Connected GitHub/GitLab accounts |
| `okctl aws` / `clusters` / `registries` | Your own cloud account |
| `okctl setup agent` / `mcp` / `skills` | Connect AI coding agents |
| `okctl open` / `docs` | Open an app in the dashboard / the docs |
| `okctl api <method> <path>` | Raw authenticated API request |
| `okctl upgrade` | Upgrade okctl to the latest release |
| `okctl completion <shell>` | Generate shell completion (bash, zsh, fish, powershell) |
| `okctl version` | Print version info |

Run `okctl <command> --help` for details on flags and arguments.

## Shell completion

```sh
# zsh
okctl completion zsh > "${fpath[1]}/_okctl"

# bash (macOS via Homebrew)
okctl completion bash > $(brew --prefix)/etc/bash_completion.d/okctl

# fish
okctl completion fish > ~/.config/fish/completions/okctl.fish
```

The Homebrew install wires completions up automatically.

## Configuration

okctl resolves settings in this order (highest priority first):

| Setting | Flag | Env var | Config file | Default |
|---|---|---|---|---|
| API URL | `--api-url` | `OKCTL_API_URL` | `api_url` in `config.yaml` | `https://app.ownkube.io` |
| Output format | `-o, --output` | — | `output_format` in `config.yaml` | `table` |
| Organization | `--organization` | `OKCTL_ORGANIZATION` | `organization` in `config.yaml` | none |
| HTTP Basic Auth | — | `OKCTL_BASIC_AUTH` (`user:pass`) | — | none |

Config files live in `~/.config/ownkube/`:

- `config.yaml` — non-sensitive preferences
- `credentials.yaml` — API key (file mode `0600`)

`OKCTL_BASIC_AUTH` is only for an API URL served behind HTTP Basic auth; `https://app.ownkube.io` authenticates with the API key alone.

## Use okctl with AI coding agents

One command sets up Claude Code, Cursor or Codex. It signs you in if needed,
adds the Ownkube MCP server, and installs the okctl skill:

```sh
okctl setup agent                  # every agent found on this machine
okctl setup agent --agent claude   # or just one: claude, cursor, codex
```

The MCP server entry runs `okctl mcp serve`, which signs the agent's requests
with your `okctl login`. No API key is written into the agent's config. To do
the two steps separately, use `okctl mcp install` and `okctl skills install`.
`okctl mcp install --print` shows the config without writing it.

The skill ([`skills/okctl/SKILL.md`](./skills/okctl/SKILL.md)) is embedded in
the binary, so it always matches the commands you have. It teaches your agent
when okctl is the right tool and which command to run for common tasks: deploys,
diagnostics, logs, variables and connection details. Re-run
`okctl setup agent` after `okctl upgrade`. You can also install it into any
[skills.sh](https://skills.sh)-compatible agent:

```sh
# Into the current project
npx skills add ownkube/ownkube-cli

# Globally (available across all projects)
npx skills add ownkube/ownkube-cli -g

# Target a specific agent
npx skills add ownkube/ownkube-cli -a claude-code
```

Then ask your agent things like *"deploy my app to Ownkube"*, *"why is my Ownkube deployment failing?"*, or *"connect to my Ownkube Postgres from this pod"* — it will use the skill to drive `okctl` for you.

## Development

```sh
make build      # → bin/okctl
make test
make lint       # requires golangci-lint
make generate   # regenerate the API client from api/openapi.json (requires oapi-codegen)
```

See [CLAUDE.md](./CLAUDE.md) for architecture notes, the subpackage layout, and the spec-regeneration workflow.

## Releasing

Tags matching `v[0-9]+.[0-9]+.[0-9]+` trigger the [release workflow](.github/workflows/release.yaml):

```sh
git tag v0.2.0
git push origin v0.2.0
```

[GoReleaser](https://goreleaser.com) builds cross-platform binaries, creates a GitHub Release, and pushes an updated formula to [ownkube/homebrew-tap](https://github.com/ownkube/homebrew-tap).

## License

[MIT](./LICENSE) © Ownkube
