# uniShell

A portable, single-binary shell experience for Linux systems.

## Project status

This repository is a clean rewrite of the original uniShell project.

The original project is retained only as a reference for behavior, ideas, and historical requirements. Its implementation is not the foundation of this project.

## Goals

- Provide a self-contained Linux executable.
- Avoid requiring the target machine's package manager.
- Bundle the required shell tooling and configuration into the release artifact.
- Support multiple Linux architectures through native CI builds.
- Keep runtime files isolated from standard system paths.
- Provide a simple CLI for shell startup and lifecycle management.
- Build reproducible release artifacts through GitLab CI/CD.
- Mirror the source repository to GitHub.

## Planned CLI

```text
unishell
unishell shell
unishell install
unishell update
unishell clean
unishell doctor
unishell version
```

