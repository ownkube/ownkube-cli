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

# Check who you are
okctl status

# Connect an AWS account (opens the console to deploy a scoped access role)
okctl aws connect

# List clusters / environments / deployments
okctl clusters list
okctl environments list
okctl deploy list --cluster <cluster-id>

# Tail logs from a deployment
okctl deploy logs <deployment-id>

# Get in-cluster connection details (namespace, service, secret)
okctl deploy connection <deployment-id>
```

Use `--output json` (or `yaml`) on any read command for machine-readable output:

```sh
okctl deploy get <deployment-id> -o json | jq .
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
| `okctl login` / `logout` / `status` | Browser-based auth + credential management |
| `okctl aws connect\|list\|get\|verify\|reconnect\|resync\|delete` | Connect an AWS account and manage its access |
| `okctl config get\|set\|view` | Manage `~/.config/ownkube/config.yaml` |
| `okctl clusters list\|get` | Inspect clusters |
| `okctl environments list\|get` | Inspect environments |
| `okctl organizations list` | List organizations you belong to |
| `okctl registries list\|get` | Inspect container registries |
| `okctl deploy list\|get\|status\|logs\|revisions\|connection` | Inspect deployments |
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
| HTTP Basic Auth (dev only) | — | `OKCTL_BASIC_AUTH` (`user:pass`) | — | none |

Config files live in `~/.config/ownkube/`:

- `config.yaml` — non-sensitive preferences
- `credentials.yaml` — API key (file mode `0600`)

`OKCTL_BASIC_AUTH` is **only** for development environments sitting behind an HTTP Basic gateway; production (`https://app.ownkube.io`) authenticates with the API key alone.

## Use okctl with AI coding agents

A Claude / Cursor / Codex / Copilot / Cline skill ships in this repo at [`skills/okctl/SKILL.md`](./skills/okctl/SKILL.md). It teaches your agent when okctl is the right tool (BYOC PaaS, cloud-credit deploys, self-hosted marketplace) and exactly which command to run for common tasks (deploy diagnostics, logs, connection details, name → ID lookups).

Install it into any [skills.sh](https://skills.sh)-compatible agent:

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
