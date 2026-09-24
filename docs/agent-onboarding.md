# uniShell Agent Onboarding

This guide gives coding agents a quick, current map of the repository. The
repository-wide working rules are in [AGENTS.md](../AGENTS.md). For fuller
project vision and operational detail, see [README.md](../README.md) and the
focused docs listed below.

## Purpose and scope

uniShell provides a curated Linux shell environment in a portable release
binary, without requiring the target system's package manager or a system-wide
installation of the bundled tools. An ephemeral launch prepares a managed
runtime for the session and cleans it up when a direct shell exits. Detaching
from a multiplexer preserves its session for reattachment; managed sessions can
be inspected and cleaned explicitly. `unishell install` provides an optional
persistent desktop setup that retains runtime data across launches.

This is a Go rewrite of an older Bash project. The old implementation is
historical reference material; do not assume its architecture, layout, or
security model applies here. The current release target is Linux amd64. Current
shells are Bash, Zsh, Fish, and Nushell. Current multiplexers are tmux and
Zellij.

## Source map

| Area | Location | Responsibility |
|---|---|---|
| User-facing program | `cmd/unishell/` | CLI, command handling, and startup orchestration |
| Runtime bundle tools | `cmd/bundle-builder/`, `cmd/tool-fetch/` | Bundle creation and tool acquisition |
| Application lifecycle | `internal/app/` | Runtime and managed-session coordination |
| Bundle and runtime | `internal/bundle/`, `internal/crypto/`, `internal/runtime/` | Bundle formats, authentication, extraction, runtime paths and cleanup |
| Credentials and install state | `internal/credentials/`, `internal/persistence/` | Token input/storage and persistent installation configuration |
| Shells | `internal/shell/` | Shell selection and process startup |
| Shell configuration | `internal/shell/config/`, `internal/shell/profile/`, `assets/config/shell/` | Shared and shell-specific startup configuration |
| Multiplexers | `internal/multiplexer/`, `assets/config/tmux/`, `assets/config/zellij/` | tmux/Zellij sessions and configuration |
| Tool acquisition | `internal/acquisition/`, `assets/tools/`, `assets/bin/` | Tool metadata, downloads, validation, and build inputs |
| Build and CI | `scripts/`, `.gitlab-ci.yml`, `.github/workflows/` | Local build, validation, release preparation, and CI |

Tests are colocated with the Go packages. Start with the package closest to the
behavior being changed, then inspect its callers and tests.

## Runtime and bundle notes

The build process stages runtime assets, builds a bundle, and uses the
`unishell_bundle` build tag to embed it in release binaries. A regular Go build
without that tag is useful for compiling the CLI but does not include a
production runtime bundle.

Version 4 bundles currently gate use with an HMAC-SHA256 token check, while the
compressed runtime payload remains plaintext and is not covered by that tag.
Versions 2 and 3 are retained for compatibility. Treat changes to bundle
encoding, credentials, extraction, and cleanup as security-sensitive, and do
not claim confidentiality or payload integrity for version 4.

## Common local commands

```bash
go test ./...
go vet ./...
scripts/ci/validate.sh
go build -o bin/unishell ./cmd/unishell
```

`scripts/ci/validate.sh` runs formatting checks, Go tests, `go vet`, and a diff
whitespace check. The release build script is `scripts/build.sh`; it may
download tool artifacts, use credentials, and create release binaries. Read
the script and select the appropriate flags before using it.

## Documentation by task

- Runtime/environment configuration: [environment.md](environment.md)
- Adding or changing bundled tools: [tool-acquisition.md](tool-acquisition.md)
- Project vision, behavior, and design history: [README.md](../README.md)
- Branch and release process: [git-workflow.md](git-workflow.md)
- Nexus synchronization context: [nexus-sync.md](nexus-sync.md)

Some documentation records design intent, old plans, or deployed operations
that are outside this checkout. Check both the referenced path and current
implementation before treating it as an executable procedure.
