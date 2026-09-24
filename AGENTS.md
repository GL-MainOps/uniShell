# Instructions for AI Agents

This file is the repository-wide starting point for coding agents. Read it
before making changes, then use [the project onboarding guide](docs/agent-onboarding.md)
to find the relevant source and documentation.

## Project in brief

uniShell is a Go application that packages a curated Linux shell environment
and its tools into release binaries. It supports ephemeral launches and an
optional persistent desktop installation. The current shell implementations
include Bash, Zsh, Fish, and Nushell; the supported multiplexers include tmux
and Zellij. The current release target is Linux amd64.

The repository is licensed for personal, private, non-commercial use under
the terms in [LICENSE](LICENSE). Read the license before proposing or
describing reuse, distribution, or commercialization.

## Before editing

- Check `git status --short --branch` first. The checkout may contain user
  changes; preserve them. Do not reset, clean, overwrite, or reformat unrelated
  files.
- Read the relevant implementation and its nearby tests before changing
  behavior. Go tests live alongside their packages.
- Keep changes focused on the request. Avoid drive-by refactors and generated
  output changes unless they are needed for the task.
- Treat source code and tests as the authority for current behavior. The README
  is useful for purpose and design context, but some architecture sections
  describe earlier plans. Check the code before relying on those sections.
- The README's detailed AI section includes process guidance from an earlier
  workflow. Follow the current user's request for whether to explain, pause, or
  commit; do not add approval gates to work the user has already authorized.
- Do not create or modify secrets. In particular, never print, commit, or put
  `UNISHELL_AUTH_TOKEN`, `UNISHELL_GITHUB_TOKEN`, Nexus credentials, or other
  private values in fixtures, logs, or documentation.

## Architecture map

- `cmd/unishell`: CLI parsing, command dispatch, and application orchestration.
- `cmd/bundle-builder`: builds a runtime bundle for embedding.
- `cmd/tool-fetch`: acquires and validates configured tool artifacts.
- `cmd/bundle-benchmark`: bundle performance utility.
- `internal/app`: application lifecycle and session coordination.
- `internal/credentials`, `internal/crypto`: credential input/storage and
  versioned bundle authentication/compatibility.
- `internal/bundle`, `internal/runtime`, `internal/session`: bundle loading and
  extraction, runtime paths/lifecycle, and managed session metadata/processes.
- `internal/acquisition`: tool metadata, downloads, caching, and validation.
- `internal/shell`, `internal/shell/config`, `internal/shell/profile`,
  `internal/shellargs`: shell selection, startup, configuration, profiles, and
  command argument handling.
- `internal/multiplexer`: tmux/Zellij integration, discovery, and session
  management.
- `internal/persistence`: installed-runtime configuration and credential
  persistence.
- `assets/config`, `assets/scripts`, `assets/tools`: runtime configuration,
  scripts, and TOML tool definitions. `assets/bin` is also used for acquired
  tool inputs; most of its contents are ignored/generated build inputs.
- `scripts/`: build, validation, and release helpers. CI definitions are in
  `.gitlab-ci.yml` and `.github/workflows/`.

## Security-sensitive areas

Changes to bundle formats, credential handling, runtime extraction, path
validation, permissions, session cleanup, or process startup need particular
care. Inspect the relevant implementation and tests, preserve compatibility
unless the task calls for a change, and describe security properties precisely.

Current version 4 bundles use an HMAC-SHA256 token gate. Their compressed
payload is plaintext in the bundle, and the version 4 tag does not authenticate
that payload. Do not describe this format as encryption or as payload integrity
protection. Version 2 and 3 formats remain supported for compatibility; verify
the implementation before changing these guarantees.

## Build and validation

For Go changes, use the checks relevant to the change:

```bash
go test ./...
go vet ./...
```

The repository CI validation entry point is:

```bash
scripts/ci/validate.sh
```

A plain CLI build is:

```bash
go build -o bin/unishell ./cmd/unishell
```

The release-oriented `scripts/build.sh` acquires or consumes tool artifacts,
builds bundles, and writes release binaries. It can use network access and
configured credentials; inspect its flags and effects before running it. Do
not run a release, publish artifacts, push, tag, or commit unless the user
requested that action.

## Documentation guide

- [README.md](README.md): project vision, runtime behavior, architecture goals,
  and the existing AI development contract. Confirm implementation details in
  code when a section may describe an earlier plan.
- [docs/agent-onboarding.md](docs/agent-onboarding.md): concise project map,
  current implementation boundaries, and task-to-document pointers.
- [docs/environment.md](docs/environment.md): environment variables and
  configuration precedence.
- [docs/tool-acquisition.md](docs/tool-acquisition.md): tool definitions,
  acquisition, profiles, and validation.
- [docs/git-workflow.md](docs/git-workflow.md): the documented branch and
  release workflow; follow the user's requested workflow for the current task.
- [docs/nexus-sync.md](docs/nexus-sync.md): deployment/synchronization context;
  check which referenced scripts and service files are actually present before
  acting on it.
