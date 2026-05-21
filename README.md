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

## Commands

| Group | What it does |
|---|---|
| `okctl login` / `logout` / `status` | Browser-based auth + credential management |
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
| API URL | `--api-url` | `OKCTL_API_URL` | `apiUrl` in `config.yaml` | `https://app.ownkube.io` |
| Output format | `-o, --output` | — | `outputFormat` in `config.yaml` | `table` |
| HTTP Basic Auth | — | `OKCTL_BASIC_AUTH` (`user:pass`) | — | none |

Config files live in `~/.config/ownkube/`:

- `config.yaml` — non-sensitive preferences
- `credentials.yaml` — API key (file mode `0600`)

`OKCTL_BASIC_AUTH` is intended for development environments behind an HTTP gateway; it adds an `Authorization: Basic …` header to every request alongside the API key.

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
