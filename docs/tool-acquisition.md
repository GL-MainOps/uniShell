# uniShell Tool Acquisition Guide

This document explains how to add a tool that uniShell should acquire during
the build process.

This is a contributor/developer document. It describes the current
declarative acquisition contract and is intended to be understandable both
to human contributors and to automated development agents such as AI/LLM
coding assistants.

## 1. Purpose

uniShell acquires external executable artifacts during the build process so
that they can eventually become part of the protected runtime bundle.

The acquisition system is declarative:

```text
tool definition
      |
      v
assets/tools/<tool-name>.toml
      |
      v
metadata loading
      |
      v
artifact resolution
      |
      v
download / cache
      |
      v
checksum verification
      |
      v
staging / extraction
      |
      v
binary selection
      |
      v
canonical binary name
      |
      v
ELF validation
      |
      v
future musl / executable validation
      |
      v
final promotion
      |
      v
assets/bin/<binary>
      |
      v
runtime bundle
```

Do not add tool-specific download logic to Go code when an existing  
acquisition source type can describe the artifact.

A new tool should normally be represented by a new TOML definition under:

```text
assets/tools/
```

## 2. Production File Location

Production tool definitions belong in:

```text
assets/tools/<tool-name>.toml
```

Example:

```text
assets/tools/zellij.toml
assets/tools/fzf.toml
assets/tools/fd.toml
```

The filename should identify the tool clearly.

The file in:

```text
docs/examples/tool-acquisition.example.toml
```

is only a blueprint. It is not a production tool definition.

Do not place production tool definitions in `docs/examples/`.

Tool-definition files under `assets/tools/` are build-time acquisition  
metadata. They are not runtime assets and are intentionally excluded from  
the generated runtime bundle.

## 3. Basic Structure

A tool definition has this structure:

```toml
[[tools]]
name = "<tool-name>"

[[tools.artifacts]]
version = "<tool-version>"
platform = "linux"
architecture = "amd64"
archive_type = "<archive-type>"
binary_path = "<path-inside-archive-or-empty-for-direct-artifact>"
binary_name = "<canonical-executable-name>"
checksum = "<64-hexadecimal-characters>"

[tools.artifacts.validation]
static_elf = true
musl = true
executable = true

[tools.artifacts.source]
kind = "<source-kind>"

[tools.artifacts.source.<source-kind>]
# source-specific fields
```

The exact fields supported by the current metadata model are:

```text
ToolMetadata
  name
  artifacts[]

ArtifactMetadata
  version
  platform
  architecture
  archive_type
  binary_path
  binary_name
  checksum
  validation
  source

SourceMetadata
  kind
  github_release
  github_file
  direct_url
```

Unknown TOML fields are rejected by the metadata loader. Therefore, do not  
invent field names in a tool definition.

## 4. Tool Name

The tool name identifies the logical tool.

Example:

```toml
[[tools]]
name = "example"
```

For a real tool:

```toml
[[tools]]
name = "zellij"
```

The tool name should be stable and correspond to the executable/tool being  
introduced.

## 5. Artifact Definitions

A tool may have one or more artifacts:

```toml
[[tools.artifacts]]
version = "1.0.0"
platform = "linux"
architecture = "amd64"
...
```

An artifact describes one concrete downloadable build.

This allows the same logical tool to eventually have different artifacts  
for different supported platforms or architectures.

The current uniShell target is:

```text
linux/amd64
```

Do not add another architecture merely because an upstream project publishes  
one. Additional architecture support must be explicitly introduced into the  
uniShell project.

## 6. Version

`version` identifies the upstream artifact version.

Example:

```toml
version = "1.0.0"
```

Use the actual version represented by the downloaded artifact.

Do not use a version that does not correspond to the artifact being acquired.

## 7. Platform

`platform` identifies the target operating system.

Current target:

```toml
platform = "linux"
```

The current uniShell release target is:

```text
linux/amd64
```

## 8. Architecture

`architecture` identifies the target CPU architecture.

Current target:

```toml
architecture = "amd64"
```

Do not expand architecture support as part of adding an individual tool.

## 9. Archive Type

`archive_type` describes how the downloaded artifact is packaged.

For a direct executable:

```toml
archive_type = ""
```

For an archive, use one of the archive types currently supported by the  
staging implementation.

Current supported archive types are:

```text
tar
tar.gz
tgz
zip
```

Example:

```toml
archive_type = "tar.gz"
```

Do not assume that another archive format is supported merely because an  
upstream project publishes it.

If a required format is not supported, that is an acquisition-framework  
change and must be implemented separately before the tool definition depends  
on it.

## 10. Binary Path

For an archived artifact, `binary_path` identifies the executable inside
the extracted archive.

The path is relative to the artifact staging root.

For a direct executable artifact (`archive_type = ""`), `binary_path` is not
used to locate the executable. In that case, `binary_name` identifies the
canonical executable path created in the staging root.

Example:

```toml
binary_path = "example-1.0.0-linux-amd64/example"
```

For an archive containing:

```text
example-1.0.0-linux-amd64/
├── example
└── README
```

the corresponding definition is:

```toml
binary_path = "example-1.0.0-linux-amd64/example"
```

For a direct executable artifact, do not provide a meaningful archive path in
`binary_path`. The canonical executable is created using `binary_name`.

Do not use an absolute path.

Do not use a path that escapes the staging root.

Unsafe examples:

```toml
binary_path = "/tmp/example"
binary_path = "../example"
binary_path = "../../bin/example"
```

Archive extraction and staging enforce filesystem-root containment.

## 11. Binary Name

`binary_name` is the canonical executable name used by uniShell.

Example:

```toml
binary_name = "example"
```

The downloaded filename does not determine the final executable name.

For an archive containing:

```text
example-1.0.0-linux-amd64/example
```

the canonical name can still be:

```toml
binary_name = "example"
```

The staging layer uses this value to normalize the executable location.

The value must be a valid relative path within the staging root and must not  
escape that root.

Unsafe examples:

```toml
binary_name = "../example"
binary_name = "/tmp/example"
```

A missing `binary_name` is invalid.

For normal tools, use a simple executable filename such as:

```toml
binary_name = "zellij"
binary_name = "fzf"
binary_name = "fd"
```

## 12. Checksum

Every production artifact must provide its expected SHA-256 checksum.

Current format:

```toml
checksum = "<64-hexadecimal-characters>"
```

Example shape:

```toml
checksum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
```

The checksum must represent the downloaded artifact itself.

Do not calculate the checksum from an extracted executable when the  
acquisition system is verifying the downloaded archive.

Do not copy a checksum from an unrelated release.

The checksum is a supply-chain verification boundary:

```text
download
   |
   v
checksum verification
   |
   +-- failure --> reject artifact
   |
   +-- success --> continue acquisition
```

Never intentionally bypass checksum verification for a production tool.

## 13. Validation Requirements

Validation requirements describe properties expected from the acquired
executable.

Current fields include:

```toml
[tools.artifacts.validation]
static_elf = true
musl = true
executable = true
```

These requirements are evaluated by the acquisition validation pipeline after
the artifact has been staged and canonicalized.

### `static_elf`

When:

```toml
static_elf = true
```

the artifact must be a valid ELF executable and must not contain:

- an ELF interpreter (`PT_INTERP`);

- a shared-library dependency represented by a `DT_NEEDED` entry.


Both traditional static ELF executables (`ET_EXEC`) and static PIE executables
(`ET_DYN`) are accepted.

A `PT_DYNAMIC` program segment by itself does not cause rejection. The
validator inspects its dynamic entries and rejects the artifact only when a
`DT_NEEDED` dependency is present.

The current static ELF validator uses Go's ELF implementation directly rather
than depending on external host commands such as `file`, `readelf`, or `ldd`.

### `musl`

When:

```toml
musl = true
```

the artifact is expected to satisfy uniShell's musl validation contract.

The musl validation stage is intentionally non-restrictive with respect to
ELF metadata that cannot reliably identify the libc implementation.

The validator MUST NOT reject an artifact merely because:

- the ELF file is stripped;
- musl-specific symbols are absent;
- the ELF file contains a `PT_DYNAMIC` segment;
- the ELF file does not expose an unambiguous libc identity;
- the artifact filename or download URL does not contain `musl`.

The validator currently verifies that the staged artifact is a valid executable
ELF of a supported type. It does not claim to prove the libc implementation.

The validator MUST NOT treat the absence of a detectable musl marker as proof
that the artifact is not musl.

The current validator does not reject an artifact solely because musl identity
cannot be established from the ELF metadata available to the validator.

The `musl` requirement therefore acts as a validation boundary without making
fragile ELF fingerprints a release blocker.

Manual validation of newly introduced tools across the project's supported
Linux environments remains part of the contributor acceptance process.

Do not treat the filename or download URL containing `musl` as proof that the
artifact satisfies this requirement.

### `executable`

When:

```toml
executable = true
```

the artifact is expected to satisfy uniShell's executable validation contract.

Executable-property validation is introduced by a later acquisition-framework
stage.

Independently of the artifact validation requirements, the final installation
step explicitly sets the promoted binary's permissions to:

```text
0755
```

Therefore, an upstream artifact that does not have the executable bit set can
still be promoted into `assets/bin` as an executable.

This permission normalization occurs at the final promotion boundary. It does
not replace artifact validation.

### Validation order

The current acquisition flow is:

```text
source resolution
      |
      v
download / cache
      |
      v
checksum verification
      |
      v
staging / extraction
      |
      v
binary selection
      |
      v
canonicalization
      |
      v
validation
      |
      v
final promotion
```

Validation failures prevent the artifact from being returned by the
acquisition pipeline and therefore prevent it from being promoted into
`assets/bin`.

Do not treat "the metadata loaded successfully" as equivalent to "the tool is
ready for release."

## 14. Source Types

The current metadata model supports these source kinds:

```text
github-release
github-file
direct-url
```

Use an existing source kind whenever it can represent the upstream artifact.

### GitHub Release

For an artifact published as a GitHub release asset:

```toml
[tools.artifacts.source]
kind = "github-release"

[tools.artifacts.source.github_release]
owner = "<github-owner>"
repository = "<github-repository>"
release = "<release-selector>"
asset = "<asset-filename>"
```

Example structure:

```toml
owner = "example"
repository = "example"
release = "latest"
asset = "example-1.0.0-linux-amd64.tar.gz"
```

The release selector and asset name must identify the intended upstream  
artifact.

### GitHub File

For a file published directly in a GitHub repository:

```toml
[tools.artifacts.source]
kind = "github-file"

[tools.artifacts.source.github_file]
owner = "<github-owner>"
repository = "<github-repository>"
path = "<repository-path>"
ref = "<git-ref>"
```

### Direct URL

For an artifact available directly from a URL:

```toml
[tools.artifacts.source]
kind = "direct-url"

[tools.artifacts.source.direct_url]
url = "https://example.invalid/example.tar.gz"
```

Use the actual upstream URL.

Do not use a temporary personal mirror unless the project explicitly  
requires that source.

## 15. `tool-fetch` Build Utility

`tool-fetch` is the build-time command used to acquire the external tools
described by the production definitions under:

```text
assets/tools/
```

It is implemented by:

```text
cmd/tool-fetch/
```

The resulting binary is generated by:

```text
scripts/build.sh
```

### Building `tool-fetch`

The normal build process builds `tool-fetch` with:

```bash
go build \
    -trimpath \
    -ldflags "-s -w" \
    -o bin/tool-fetch \
    ./cmd/tool-fetch
```

The generated binary is a build artifact under:

```text
bin/tool-fetch
```

It is not itself part of the runtime bundle.

The build script then invokes it to acquire the runtime tools before the
runtime assets are copied into the temporary bundle source directory.

### Normal build flow

The normal `scripts/build.sh` flow is:

```text
scripts/build.sh
      |
      v
build bin/tool-fetch
      |
      v
run bin/tool-fetch
      |
      v
load assets/tools/*.toml
      |
      v
acquire / cache / stage / validate tools
      |
      v
promote executables into assets/bin/
      |
      v
build bundle-builder
      |
      v
copy assets/ into temporary runtime directory
      |
      v
generate runtime bundle
      |
      v
build unishell
```

The tool-definition directory is intentionally excluded from the generated
runtime bundle.

### Skipping tool acquisition

The build script accepts:

```text
--skip-fetch
```

Example:

```bash
scripts/build.sh --skip-fetch
```

When this option is supplied, `scripts/build.sh` does not build or invoke
`tool-fetch`. It proceeds using the runtime tools already present in
`assets/bin/`.

`--skip-fetch` is a flag of `scripts/build.sh`, not a flag of `tool-fetch`.

### `tool-fetch` command-line flags

`tool-fetch` accepts the following flags:

| Flag             | Default                 | Purpose                                                |
| ---------------- | ----------------------- | ------------------------------------------------------ |
| `--tools-dir`    | `assets/tools`          | Directory containing tool acquisition TOML definitions |
| `--output-dir`   | `assets/bin`            | Directory where acquired executables are installed     |
| `--cache-dir`    | `tmp/acquisition-cache` | Directory used for acquired artifact cache             |
| `--platform`     | `linux`                 | Target platform                                        |
| `--architecture` | `amd64`                 | Target architecture                                    |

Example:

```bash
bin/tool-fetch
```

Equivalent to:

```bash
bin/tool-fetch \
    --tools-dir assets/tools \
    --output-dir assets/bin \
    --cache-dir tmp/acquisition-cache \
    --platform linux \
    --architecture amd64
```

A different target can be selected explicitly:

```bash
bin/tool-fetch \
    --platform linux \
    --architecture amd64
```

The current uniShell target is `linux/amd64`.

### Tool acquisition behavior

For every tool loaded from the selected tool-definition directory,
`tool-fetch`:

1. resolves the artifact matching the requested platform and architecture;
2. acquires the artifact through the configured acquisition provider;
3. uses the filesystem cache under the configured cache directory;
4. stages or extracts the artifact under an acquisition staging directory;
5. selects the declared executable;
6. canonicalizes it using the declared `binary_name`;
7. validates the staged artifact;
8. promotes the canonical executable into the configured output directory;
9. removes the staging directory.

The staging directory is derived from the cache directory:

```text
<cache-parent>/acquisition-stage/

With the default cache directory:
```

```text
tmp/acquisition-cache
```

the staging directory is:

```text
tmp/acquisition-stage
```
### Final executable promotion

The final promotion is performed atomically through a temporary file in the
destination directory followed by a rename.

The promoted executable is always assigned:

```text
0755
```

This is intentional.

The permissions of the upstream artifact do not determine the permissions of
the final executable under `assets/bin/`.

For example, an upstream artifact with:

```text
0644
```

is still promoted as:

```text
0755
```

This normalization is a final-installation safeguard and is separate from
artifact validation.

### Output

With the default configuration, acquired tools are placed directly under:

```text
assets/bin/
```
using their canonical `binary_name`.

For example:

```text
assets/tools/zellij.toml
        |
        v
acquisition
        |
        v
assets/bin/zellij
```

`assets/bin/` is a build-time output location. The generated runtime bundle
later incorporates the required runtime assets from the `assets/` tree.

### `tool-fetch` and production tool definitions

`tool-fetch` does not contain tool-specific download logic.

To add a supported tool, normally add its declarative definition under:

```text
assets/tools/<tool-name>.toml
```
Do not modify `cmd/tool-fetch/main.go` merely to register another tool when
the existing acquisition framework can represent it declaratively.

Changes to the acquisition framework itself should be treated separately from
adding an individual tool definition.

## 16. Complete Example

The following is a generic example of an archived Linux amd64 executable:

```toml
[[tools]]
name = "example"

[[tools.artifacts]]
version = "1.0.0"
platform = "linux"
architecture = "amd64"
archive_type = "tar.gz"
binary_path = "example-1.0.0-linux-amd64/example"
binary_name = "example"
checksum = "replace-with-64-hexadecimal-characters"

[tools.artifacts.validation]
static_elf = true
musl = true
executable = true

[tools.artifacts.source]
kind = "github-release"

[tools.artifacts.source.github_release]
owner = "example"
repository = "example"
release = "latest"
asset = "example-1.0.0-linux-amd64.tar.gz"
```

This example is a template. Do not copy its placeholder values into a  
production tool definition.

## 17. Adding a New Tool: Contributor Procedure

When adding a new tool, follow this sequence.

### Step 1 — Identify the exact upstream artifact

Determine:

```text
tool name
version
platform
architecture
download source
downloaded filename
archive format
executable path inside archive
```

Do not create the TOML definition until these values are known.

### Step 2 — Confirm source compatibility

Determine whether the artifact can be represented by one of:

```text
github-release
github-file
direct-url
```

If none applies, stop and treat the missing source type as an acquisition  
framework change rather than inventing a new TOML field.

### Step 3 — Determine the archive layout

If the artifact is an archive, inspect its contents and identify the exact  
relative path to the executable.

Set:

```toml
binary_path = "<relative-path-to-executable>"
```

Do not guess the archive layout from the filename.

### Step 4 — Determine the canonical executable name

Choose the executable name that uniShell should expose:

```toml
binary_name = "<executable-name>"
```

Normally this should be the command users expect to execute.

### Step 5 — Obtain the artifact checksum

Calculate or obtain the authoritative SHA-256 checksum for the exact
downloaded artifact.

Set:

```toml
checksum = "<64-hexadecimal-characters>"
```

Do not substitute a checksum from another version, architecture, or asset.

### Step 6 — Declare validation requirements

Set the validation requirements expected for the artifact:

```toml
[tools.artifacts.validation]
static_elf = true
musl = true
executable = true
```

These requirements must describe the actual artifact.

### Step 7 — Create the production definition

Create:

```text
assets/tools/<tool-name>.toml
```

Do not modify the Go acquisition implementation merely to register the  
tool if the existing declarative model already supports it.

### Step 8 — Validate the definition

Run the acquisition metadata tests and the broader project validation.

At minimum, verify:

```text
metadata parsing
metadata validation
artifact conversion
acquisition tests
full project tests
go vet
git diff --check
```

For an artifact that is actually downloaded, also verify the complete  
acquisition/staging path.

### Step 9 — Verify the artifact itself

A successful TOML parse does not prove that the upstream artifact is usable.

The artifact must eventually pass the acquisition validation pipeline:

```text
source resolution
      |
      v
download
      |
      v
checksum verification
      |
      v
archive extraction
      |
      v
binary selection
      |
      v
canonicalization
      |
      v
ELF validation
      |
      v
musl/static validation
      |
      v
executable validation
```

Do not treat "the metadata loaded successfully" as equivalent to "the tool  
is ready for release."

## 18. AI/LLM Contributor Rules

An AI/LLM modifying uniShell should follow these rules when adding a tool.

1. Inspect the current acquisition implementation before changing metadata.
    
2. Treat the current Go structs and validation code as the authoritative  
    implementation contract.
    
3. Do not invent TOML fields.
    
4. Do not invent unsupported source types.
    
5. Do not assume an archive format is supported.
    
6. Do not assume the executable path inside an archive.
    
7. Inspect or otherwise verify the upstream artifact layout.
    
8. Use the exact artifact checksum.
    
9. Keep `binary_path` and `binary_name` distinct:
    
    - `binary_path` identifies the executable in the downloaded archive.
        
    - `binary_name` identifies the canonical executable name used by uniShell.
        
10. Never use an absolute executable path.
    
11. Never use a path that escapes the staging root.
    
12. Do not add tool-specific acquisition code when the existing declarative  
    model is sufficient.
    
13. Do not modify unrelated acquisition infrastructure.
    
14. Add tests when the requested tool exposes a previously unsupported  
    acquisition requirement.
    
15. Run the relevant tests before claiming the tool is valid.
    
16. Run `go test ./...`.
    
17. Run `go vet ./...`.
    
18. Run `git diff --check`.
    
19. Review the final diff for unrelated changes.
    
20. Follow the project's incremental checkpoint and Conventional Commit  
    workflow.
    

If the tool requires a capability that the current acquisition framework  
does not support, do not silently redesign the framework. Report the missing  
capability and make it a separate implementation checkpoint.

## 19. What Adding a Tool Does Not Do

Adding a file under:

```text
assets/tools/
```

defines an acquisition artifact.

It does not by itself mean that the tool is:

- exposed through the user-facing CLI;
    
- added to the shell PATH;
    
- integrated into a shell configuration;
    
- included in the final runtime bundle;
    
- validated as a static ELF;
    
- validated as musl-linked;
    
- selected as a runtime dependency.
    

Those concerns belong to later stages of the roadmap.

The acquisition framework is intentionally being stabilized before concrete  
runtime population.

## 20. Current Roadmap Relationship

The acquisition framework belongs to Phase B of the uniShell roadmap.

The intended order is:

```text
B1  Tool Acquisition Framework
B2  Declarative Artifact Metadata
B3  Download / Cache
B4  Checksum Verification
B5  Archive Extraction
B6  Static ELF Validation
B7  musl Validation
B8  Executable Validation
B9  Platform / Architecture Validation
B10 Acquisition Tests
     |
     v
Phase C
     |
     +-- C1 Zellij Acquisition
     +-- C2 tmux Acquisition
     +-- C3 Shell Binaries
     +-- C4 Core Shell Tools
     +-- C5 Asset Manifest
```

The current document describes how contributors use the declarative  
acquisition system. It does not replace the implementation roadmap.

## 21. Source of Truth

When this document conflicts with the implementation, do not silently  
choose one.

The current implementation and tests define the actual behavior of the  
software.

A documentation mismatch should be treated as a documentation or  
implementation issue and resolved explicitly.

When an acquisition capability changes, update this document as part of the  
same appropriate documentation checkpoint.

