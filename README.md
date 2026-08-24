# uniShell

A portable, single-binary Linux shell-experience runtime designed to
provide a curated set of shell tools without depending on the target
machine's package manager or requiring permanent installation of those
tools into standard system paths.

## License

uniShell is proprietary software.

It is licensed exclusively for personal, private, non-commercial use
under the `uniShell Personal Use License`.

See [LICENSE](./LICENSE).

Commercial use, redistribution, modification, derivative works,
sublicensing, and forking are not permitted unless separately authorized
in writing by the copyright holder.

---

# 1. Project Vision

uniShell is being rebuilt completely from scratch.

The previous uniShell project is a historical reference only. Its
implementation, Bash architecture, obfuscation mechanism, directory
layout, cryptographic implementation, and deployment mechanism are NOT
the foundation of this project.

The new project should preserve useful behavioral requirements from the
old project while replacing its implementation with a maintainable,
modular, testable, and DevOps-oriented architecture.

The ultimate deployment goal is:

```text
one binary
    |
    +-- authenticate
    |
    +-- decrypt embedded runtime bundle
    |
    +-- create temporary runtime
    |
    +-- extract required tools
    |
    +-- start enhanced shell
    |
    +-- user exits shell
    |
    +-- clean temporary runtime
    |
    +-- leave only the uniShell binary
```

The target machine should not need:

- a package manager;
    
- a system-wide installation;
    
- permanently installed copies of bundled tools;
    
- configuration files in standard system locations;
    
- a permanently extracted copy of the encrypted runtime payload.
    

---

# 2. Historical Reference Project

The original project used a Bash-based implementation.

Its historical design included:

- a temporary hidden runtime directory;
    
- bundled command-line tools;
    
- an encrypted `lesscache.dat` payload;
    
- password authentication;
    
- extraction of tools into a temporary tools directory;
    
- modification of `PATH`;
    
- shell initialization through a generated/obfuscated Bash script;
    
- cleanup of extracted tools when the shell/session ended;
    
- Git hooks that generated an obfuscated representation of the main script;
    
- remote deployment through SSH-oriented workflows.
    

The old project also used tools such as:

```text
bat
fd
fzf
lsd
sysmon
tmux
```

The old implementation is retained only as behavioral and historical  
reference material.

DO NOT copy the old architecture into the new project unless explicitly  
requested and justified.

The new architecture must be designed independently.

---

# 3. Primary Design Goals

The project should prioritize the following:

1. Single-binary deployment.
    
2. Linux-first operation.
    
3. amd64 (`linux/amd64`) as the current target platform.
    
4. No dependency on the target machine's package manager.
    
5. No requirement to permanently install bundled tools.
    
6. Runtime files isolated under a configurable runtime directory.
    
7. Authentication before any uniShell operation.
    
8. Encrypted bundled runtime assets.
    
9. Temporary extraction of executable assets.
    
10. Cleanup after shell/session termination.
    
11. Recovery from stale runtimes after abnormal termination.
    
12. Minimal persistent footprint on target machines.
    
13. Modular internal architecture.
    
14. Strong automated testing.
    
15. Reproducible builds.
    
16. GitLab CI/CD as the primary CI/CD platform.
    
17. GitHub as a mirrored repository.
    
18. Native CI runners for release builds.
    
19. Clear versioned release artifacts.
    
20. Secure handling of credentials and encrypted assets.
    

---

# 4. Current Target Deployment Model

The intended deployment model is:

```text
Administrator/operator
        |
        | copy/download one binary
        v
target server
        |
        +-- unishell
        |
        | execute
        v
authentication
        |
        v
decrypt protected embedded bundle
        |
        v
create ephemeral runtime
        |
        v
extract tools/configuration
        |
        v
start enhanced shell
        |
        v
user logs out
        |
        v
cleanup ephemeral runtime
        |
        v
target server contains only the uniShell binary
```

The binary is the deployment artifact.

The runtime is reconstructed when needed.

---

# 5. Runtime Directory

The runtime directory is configurable at runtime.

Precedence:

```text
UNISHELL_RUNTIME_DIR
        |
        v
configured runtime directory
```

If the variable is not set, the default is:

```text
/var/tmp/.lesscache
```

The runtime directory must NOT be compiled into the binary as an  
immutable deployment location.

Examples:

```bash
UNISHELL_RUNTIME_DIR=/var/tmp/.lesscache ./unishell
```

```bash
UNISHELL_RUNTIME_DIR=/opt/unishell ./unishell
```

The runtime path abstraction is responsible for constructing paths.

Filesystem creation and cleanup belong to the runtime/session layer.

---

# 6. Runtime Lifecycle

The runtime is ephemeral.

Conceptual lifecycle:

```text
startup
   |
   v
authenticate
   |
   v
remove stale runtime
   |
   v
prepare session runtime
   |
   v
decrypt bundle
   |
   v
extract assets
   |
   v
start shell
   |
   v
shell exits
   |
   v
cleanup runtime
```

The runtime must not be treated as a permanent installation.

## Abnormal termination

Cleanup cannot be guaranteed for every possible system failure.

Examples that may prevent cleanup:

- SIGKILL;
    
- kernel panic;
    
- power loss;
    
- abrupt host termination;
    
- hardware failure.
    

Therefore the implementation should use two layers:

```text
normal termination
    |
    +-- immediate cleanup


next startup
    |
    +-- detect stale runtime
    |
    +-- remove stale runtime
```

This provides recovery from runtimes left behind by abnormal termination.

---

# 7. Persistent vs Ephemeral Data

The architecture must clearly distinguish:

## Persistent

Ideally:

```text
uniShell binary
```

The encrypted runtime payload should ultimately be embedded inside the  
binary rather than requiring a separate `lesscache.dat` deployment file.

## Ephemeral

Examples:

```text
extracted executables
temporary configuration
temporary sockets
temporary shell assets
```

These should be removed after the session.

The target-state goal is:

```text
server
|
└── unishell
```

rather than:

```text
server
|
├── unishell
├── lesscache.dat
├── extracted binaries
├── shell configuration
└── temporary runtime
```

---

# 8. Authentication Model

Every uniShell invocation requires authentication.

Authentication occurs BEFORE:

- help output;
    
- version output;
    
- command execution;
    
- runtime preparation;
    
- bundle extraction;
    
- shell startup.
    

Examples:

```bash
./unishell
```

```bash
./unishell version
```

```bash
./unishell help
```

```bash
./unishell doctor
```

All require authentication.

No command should print its normal output before authentication succeeds.

## Credential acquisition

The credential acquisition mechanism is:

```text
UNISHELL_AUTH_TOKEN
        |
        v
interactive hidden prompt
```

There is intentionally NO `--auth` or `--auth-token` CLI option.

The preferred non-interactive mechanism is:

```bash
UNISHELL_AUTH_TOKEN="$SECRET" ./unishell
```

Interactive use:

```bash
./unishell
```

Prompt:

```text
Enter Token:
```

The token must not be echoed.

The token must not be written to disk.

The token must not be logged.

The token must not be included in normal diagnostic output.

---

# 9. Credential Acquisition vs Authentication

These are separate concepts.

Credential acquisition means:

```text
obtain the user's secret
```

Authentication means:

```text
prove that the supplied secret can unlock the protected uniShell
payload.
```

The `internal/credentials` package handles acquisition.

The cryptographic bundle layer handles actual authentication.

A non-empty token is NOT automatically a valid token.

The intended future flow is:

```text
credential
    |
    v
cryptographic verification/decryption
    |
    +-- success --> continue
    |
    +-- failure --> Authentication Failed. Aborting...
```

---

# 10. Authentication Failure

Wrong credentials must produce:

```text
Authentication Failed. Aborting...
```

No runtime should be extracted when authentication fails.

No shell should be started.

No normal command output should be produced.

No protected payload should be written to disk.

---

# 11. Cryptographic Architecture

The current cryptographic design uses:

```text
password/token
      |
      v
Argon2id
      |
      v
256-bit key
      |
      v
AES-256-GCM
```

The encrypted bundle is versioned.

Conceptually:

```text
encrypted bundle
+--------------------------+
| magic                    |
| format version           |
| KDF parameters           |
| random salt              |
| nonce                    |
| authenticated ciphertext |
+--------------------------+
```

The password is never stored in the bundle.

AES-GCM provides authenticated encryption.

Wrong credentials or modified ciphertext must cause authentication  
failure.

The cryptographic parameters must be benchmarked against the actual  
target environment before being treated as production-final.

Do not replace the cryptographic implementation casually.

Any cryptographic modification requires:

- design review;
    
- tests;
    
- compatibility consideration;
    
- explicit commit;
    
- production/security review.
    

---

# 12. Test Credentials

`test-password` is a TEST credential only.

It exists for encrypted test fixtures.

Conceptually:

```text
test fixture
    |
    +-- encrypted using test-password
    |
    +-- test-password  --> success
    |
    +-- wrong-password --> authentication failure
```

The production credential is unrelated to `test-password`.

The production secret must never be committed to Git.

The production encrypted bundle must be generated by the release/build  
pipeline using a protected CI secret or an equivalent secure mechanism.

Never place a production credential in:

- source code;
    
- Git history;
    
- README files;
    
- test fixtures;
    
- shell scripts committed to the repository;
    
- CI configuration;
    
- command-line arguments;
    
- logs.
    

---

# 13. Repository Architecture

The project is being built as a modular Go application.

Current conceptual structure:

```text
cmd/
└── unishell/
    ├── main.go
    └── main_test.go

internal/
├── app/
│   ├── app.go
│   └── app_test.go
│
├── credentials/
│   ├── credentials.go
│   ├── credentials_test.go
│   └── errors.go
│
├── crypto/
│   ├── bundle.go
│   └── bundle_test.go
│
└── runtime/
    ├── paths.go
    ├── runtime.go
    ├── errors.go
    └── runtime_test.go
```

Additional packages will be introduced only when they represent a  
meaningful responsibility.

Avoid creating packages merely to split small amounts of code.

---

# 14. Responsibility Boundaries

## `cmd/unishell`

Responsibilities:

- CLI entry point;
    
- global argument parsing;
    
- process exit handling;
    
- user-facing error presentation;
    
- command dispatch.
    

It should remain thin.

Business logic should not accumulate in `main.go`.

## `internal/app`

Responsibilities:

- application-level initialization;
    
- shared application state;
    
- coordination of major application components.
    

It should not contain low-level filesystem or cryptographic  
implementation details.

## `internal/credentials`

Responsibilities:

- obtain authentication credentials;
    
- environment-variable lookup;
    
- interactive hidden prompt;
    
- empty-credential validation.
    

It does NOT determine whether the credential is correct.

## `internal/crypto`

Responsibilities:

- password-based key derivation;
    
- encrypted bundle encoding/decoding;
    
- authenticated encryption/decryption;
    
- cryptographic integrity validation.
    

## `internal/runtime`

Responsibilities:

- runtime path calculation;
    
- ephemeral runtime preparation;
    
- stale-runtime detection;
    
- cleanup;
    
- runtime filesystem safety.
    

## Future `internal/bundle`

Responsibilities will eventually include:

- packaging runtime assets;
    
- archive creation;
    
- extraction;
    
- archive path validation;
    
- embedded bundle handling.
    

## Future `internal/shell`

Responsibilities will eventually include:

- shell process startup;
    
- environment preparation;
    
- PATH construction;
    
- shell lifecycle;
    
- signal handling;
    
- cleanup coordination.
    

---

# 15. Runtime Security Boundary

Extracted files are considered untrusted filesystem output until their  
paths have been validated.

Archive extraction MUST prevent path traversal.

Examples that must never escape the configured runtime root:

```text
../../etc/passwd
../../../tmp/file
/absolute/path
```

The runtime package already contains the concept of:

```go
IsWithinRoot(...)
```

This must be used by future extraction logic.

Never extract an archive directly into the filesystem without validating  
each destination path.

---

# 16. Permissions

Runtime directories are intended to be private to the executing user.

The runtime should use restrictive permissions such as:

```text
0700
```

where appropriate.

Filesystem permission failures should produce actionable messages.

The user should be told that they can:

- fix permissions on the configured runtime directory; or
    
- choose another runtime directory using `UNISHELL_RUNTIME_DIR`.
    

Do not expose raw, confusing filesystem errors when a clearer  
diagnostic can be provided.

---

# 17. CLI Philosophy

The CLI should remain minimal.

Current/planned commands:

```text
unishell
unishell shell
unishell install
unishell update
unishell clean
unishell doctor
unishell version
```

All operations require authentication.

Global configuration should precede the command where applicable.

Example:

```bash
UNISHELL_RUNTIME_DIR=/opt/unishell ./unishell shell
```

Do not introduce a large CLI framework unless the project's complexity  
actually justifies it.

---

# 18. Single-Binary Goal

The final release should be a single executable.

Target concept:

```text
unishell
```

containing:

```text
application code
+
encrypted runtime bundle
+
required metadata
```

The encrypted bundle should remain encrypted inside the binary.

At runtime:

```text
binary
   |
   +-- authenticate
   |
   +-- decrypt in memory
   |
   +-- materialize temporary files
   |
   +-- execute
   |
   +-- cleanup
```

The release process should not require manually copying a collection of  
support files to the target server.

---

# 19. Build Target

Current target:

```text
linux/amd64
```

Do not add additional architectures unless explicitly requested.

The project may be architected so additional targets can be added later,  
but current CI/release work should focus on:

```text
GOOS=linux
GOARCH=amd64
```

---

# 20. Static / Portable Binary Goal

The binary should be suitable for deployment to Linux systems without  
requiring the target system's package manager.

Where practical, dependencies should be statically linked or otherwise  
contained within the release artifact.

The final release must be tested on representative target environments.

Do not assume that "build succeeds" means "deployment works."

---

# 21. CI/CD Direction

GitLab is the primary repository and CI/CD platform.

Primary repository:

```text
https://gitlab.com/mainops/uniShell
```

GitHub is a mirror:

```text
https://github.com/GL-MainOps/uniShell
```

GitLab is the source of truth for development and release automation.

The project should eventually provide:

```text
Git push
    |
    v
GitLab CI
    |
    +-- tests
    +-- lint
    +-- build
    +-- bundle
    +-- encrypt
    +-- embed
    +-- package
    +-- release
    |
    v
GitLab release artifact
```

GitHub should receive the mirrored repository and appropriate release  
artifacts according to the final release strategy.

Do not treat GitHub as the primary development source unless the project  
workflow is explicitly changed.

---

# 22. CI/CD Security

Production secrets must be supplied through CI/CD secret mechanisms.

Never:

```text
hard-code production passwords
commit production bundles
commit production credentials
print secrets in CI logs
pass production secrets as ordinary command-line arguments
```

The CI system should use masked/protected variables or an equivalent  
secure secret mechanism.

The release pipeline must make a clear distinction between:

```text
test build
```

and:

```text
production release build
```

Test credentials and production credentials must never be mixed.

---

# 23. Reproducibility

Builds should be as reproducible as practical.

Release metadata such as:

```text
version
commit
build date
```

should be injected at build time rather than hard-coded into source.

Release binaries should be traceable back to:

```text
Git commit
version/tag
CI pipeline
```

---

# 24. Testing Requirements

Every meaningful behavior change should have automated tests.

Run:

```bash
go test ./...
```

before committing.

Build:

```bash
go build -o bin/unishell ./cmd/unishell
```

after structural changes.

Security-sensitive behavior requires negative tests.

Examples:

```text
correct credential
wrong credential
modified encrypted bundle
corrupt bundle
truncated bundle
invalid archive
path traversal
permission denied
stale runtime
cleanup failure
```

Do not write tests that merely verify implementation details when the  
behavior can be tested through a public package boundary.

---

# 25. Current Authentication Test Model

The credential layer tests:

```text
environment token
empty token
valid token acquisition
```

The cryptographic layer tests:

```text
test-password -> decrypt successfully
wrong-password -> authentication failure
modified bundle -> authentication failure
invalid bundle -> invalid bundle error
```

The important distinction is:

```text
credentials
    |
    +-- acquisition

crypto
    |
    +-- actual authentication
```

---

# 26. Error Handling

Errors should be:

- actionable;
    
- contextual;
    
- safe;
    
- free of secrets.
    

Never include authentication tokens in errors.

Never include decrypted payload contents in errors.

Never include sensitive paths or data unless necessary for diagnosis.

Prefer wrapping errors with context:

```go
fmt.Errorf("initialize runtime paths: %w", err)
```

Use sentinel or typed errors where callers need to distinguish behavior.

Examples:

```text
ErrAuthenticationFailed
ErrInvalidBundle
PermissionError
```

Do not make callers parse human-readable error strings.

---

# 27. Cleanup Requirements

The final shell lifecycle must guarantee cleanup on normal exit.

Expected behavior:

```text
shell starts
    |
    +-- runtime exists
    |
shell exits
    |
    +-- extracted tools removed
    |
    +-- temporary configuration removed
    |
    +-- temporary runtime removed
```

Cleanup must also be attempted for:

- SIGINT;
    
- SIGTERM;
    
- normal child-process termination.
    

The design must acknowledge that SIGKILL, power loss, kernel failure,  
and equivalent events cannot execute user-space cleanup code.

Therefore stale-runtime cleanup on the next invocation is required.

---

# 28. Legacy Project Compatibility

Do not automatically preserve old implementation details.

The following are historical reference concepts only:

```text
lesscache.dat
bash-obfuscate
.obfuscated Bash entrypoint
GPG-encrypted tarball
OpenSSL AES-256-CBC payload
SHA-256 stored password hash
```

They may inform behavior or requirements, but they are not mandatory  
implementation choices for the rewrite.

The new project has already moved toward:

```text
Go
+
versioned runtime paths
+
ephemeral runtime sessions
+
credential acquisition
+
Argon2id
+
AES-256-GCM
+
single-binary deployment
```

Any decision to return to a legacy mechanism requires explicit  
architectural justification.

---

# 29. Development Workflow

The project is developed incrementally.

Do not implement the entire project in one change.

Preferred progression:

```text
architecture
    |
    v
runtime
    |
    v
credential acquisition
    |
    v
cryptographic bundle
    |
    v
bundle packaging
    |
    v
embedded bundle
    |
    v
runtime extraction
    |
    v
shell lifecycle
    |
    v
cleanup
    |
    v
release packaging
    |
    v
CI/CD
```

Each stage should be independently tested.

---

# 30. Commit Discipline

Every meaningful architectural step gets an explicit Git commit.

Commit messages should follow conventional commits.

Examples:

```text
chore: initialize Go project
chore: establish CLI architecture
fix: handle default shell command arguments
feat: introduce runtime path abstraction
refactor: make runtime root configurable
feat: add configurable runtime directory
test: cover runtime directory resolution
feat: add runtime preparation and permission errors
refactor: introduce ephemeral runtime sessions
feat: add authentication token resolution
feat: require authentication for all commands
refactor: remove authentication command flag
fix: improve authentication failure messaging
feat: add encrypted bundle support
```

Do not combine unrelated architectural changes into one commit.

When a step is complete:

```bash
go test ./...
go build -o bin/unishell ./cmd/unishell
git diff
git status
git commit
```

---

# 31. AI / Agent Development Contract

This section is part of the project's operating contract for AI coding  
agents.

Any AI assistant or coding agent working on uniShell MUST follow these  
rules.

## 31.1 Understand the architecture before modifying code

Do not immediately rewrite files.

First inspect:

```text
current repository state
current Git history
relevant package
relevant tests
README
architecture
```

Understand why the existing code exists before changing it.

## 31.2 Do not rebuild the legacy project

The old Bash project is reference material.

Do NOT:

- port the Bash script line-for-line;
    
- reproduce its architecture;
    
- reproduce its obfuscation pipeline;
    
- reproduce its directory layout without justification;
    
- reproduce its encryption implementation merely because it existed.
    

Implement the new architecture.

## 31.3 Never assume requirements

If a requirement is ambiguous and materially affects architecture,  
ask for clarification before implementing it.

Do not silently invent:

- supported architectures;
    
- deployment behavior;
    
- authentication semantics;
    
- persistent-file behavior;
    
- security guarantees;
    
- CI/CD providers;
    
- package-management requirements;
    
- cryptographic formats.
    

## 31.4 Preserve existing behavior

When adding a feature:

```text
new behavior
+
existing tests
+
existing functionality
```

must continue to work unless the change intentionally modifies the  
behavior.

If behavior must change, explicitly identify the reason.

## 31.5 Explain changes concisely

Before each implementation step, explain:

1. what is changing;
    
2. why it is changing;
    
3. which files are changing;
    
4. how it will be verified;
    
5. the exact commit point and commit message.
    

Do not provide unnecessary theory.

## 31.6 Provide complete file contents when requested

When the user explicitly requests complete files, provide the complete  
files rather than partial snippets.

When the user requests a small code fix, provide only the changed  
sections where practical.

## 31.7 Never silently alter unrelated code

Do not perform opportunistic refactors.

If an unrelated problem is discovered:

```text
identify it
|
v
explain it
|
v
wait for approval
```

unless fixing it is necessary to make the requested change correct or  
secure.

## 31.8 Never claim authentication when only credential acquisition exists

A non-empty password is not authentication.

Authentication requires cryptographic verification against the protected  
bundle.

## 31.9 Never expose secrets

AI agents must never:

- print production credentials;
    
- place production credentials in source code;
    
- place production credentials in README files;
    
- create committed secret fixtures using production secrets;
    
- recommend committing encrypted production payloads without explicit  
    security review;
    
- include secrets in error messages.
    

## 31.10 Treat cryptography as security-sensitive

Before changing:

```text
KDF
cipher
nonce handling
salt handling
authentication
bundle format
key length
encryption parameters
```

explain the security and compatibility implications first.

## 31.11 Test before commit

Every implementation step must specify:

```bash
go test ./...
```

and, where applicable:

```bash
go build -o bin/unishell ./cmd/unishell
```

Do not instruct the user to commit while tests are failing.

## 31.12 Explicitly identify commit checkpoints

The AI must explicitly state:

```text
GIT CHECKPOINT
```

and provide the exact commit command and commit message.

## 31.13 Keep the final artifact in mind

Every architectural decision should be evaluated against:

```text
Can this ultimately work inside one Linux amd64 binary?
```

If a proposed design introduces mandatory external runtime files,  
explain why before adopting it.

## 31.14 Do not promise impossible security properties

Never claim that:

```text
binary cannot be reverse engineered
```

or:

```text
temporary files can always be securely deleted
```

or:

```text
a secret can be guaranteed to disappear from process memory
```

Native binaries can be reverse engineered.

Cleanup cannot execute after every possible system failure.

Managed runtimes may retain sensitive memory.

Use realistic security claims.

---

# 32. AI Change Procedure

For every requested feature, follow this sequence:

```text
1. Inspect current state
       |
2. Identify affected architecture
       |
3. Explain proposed change
       |
4. Identify files
       |
5. Implement minimal change
       |
6. Add/update tests
       |
7. Run formatting
       |
8. Run tests
       |
9. Build
       |
10. Review diff
       |
11. Give commit message
       |
12. Wait for user confirmation
       |
13. Continue
```

Do not skip the testing stage.

Do not skip the commit checkpoint.

---

# 33. Preferred AI Prompt

The following prompt can be provided to an AI coding agent before it  
starts working on uniShell:

```text
You are working on the uniShell project.

Read the repository README.md and LICENSE before making changes.

uniShell is a clean rewrite of an older Bash-based project. The old
project is reference material only. Do not port or reproduce the old
implementation unless explicitly requested.

PROJECT GOAL

Build a portable Linux amd64 single-binary shell experience.

The target server should ideally require only:

    /path/to/unishell

When executed:

    1. Obtain the authentication token.
    2. Authenticate against the encrypted embedded runtime bundle.
    3. Reject incorrect credentials before any protected operation.
    4. Create an ephemeral runtime.
    5. Decrypt and extract the required tools/configuration.
    6. Start the enhanced shell.
    7. Remove the ephemeral runtime when the shell exits.
    8. Leave only the uniShell binary after normal cleanup.

RUNTIME DIRECTORY

Use:

    UNISHELL_RUNTIME_DIR

when provided.

Default:

    /var/tmp/.lesscache

Do not hard-code the deployment location into the binary.

AUTHENTICATION

Every command requires authentication, including:

    help
    version
    doctor
    clean
    shell
    install
    update

Credential acquisition:

    UNISHELL_AUTH_TOKEN
        |
        v
    hidden interactive prompt

Interactive prompt must be exactly:

    Enter Token:

There is no --auth or --auth-token option.

Never log or persist the token.

Credential acquisition is NOT authentication.

Actual authentication occurs when the supplied credential is used to
authenticate/decrypt the protected bundle.

Wrong credentials must eventually result in:

    Authentication Failed. Aborting...

No protected runtime may be extracted when authentication fails.

CRYPTOGRAPHY

The current cryptographic design is:

    Argon2id
        |
        v
    256-bit key
        |
        v
    AES-256-GCM

The encrypted bundle is versioned and contains the parameters required
for decryption, a random salt, nonce/ciphertext, and authenticated data
as appropriate.

Do not change cryptographic primitives or parameters without explaining
the security and compatibility consequences.

TEST PASSWORD

"test-password" is only a test credential.

It must never become the production credential.

Production credentials must be supplied securely through the release
process and must never be committed.

RUNTIME SECURITY

Runtime files are ephemeral.

Normal shell exit must trigger cleanup.

The implementation must also handle stale runtimes on the next startup
because SIGKILL, power loss, kernel failures, and similar events can
prevent cleanup.

Archive extraction must prevent path traversal.

Never allow:

    ../../etc/passwd
    ../../../tmp/file
    /absolute/path

to escape the configured runtime root.

PLATFORM

Current release target:

    linux/amd64

Do not add other architectures unless explicitly requested.

CI/CD

GitLab is the primary repository and CI/CD platform.

GitLab:

    https://gitlab.com/mainops/uniShell

GitHub is a mirror:

    https://github.com/GL-MainOps/uniShell

Release builds should eventually use native CI runners and produce the
single-binary release artifact.

DEVELOPMENT STYLE

Use Go.

Keep packages modular.

Keep cmd/unishell thin.

Put business logic under internal/.

Do not introduce dependencies without justification.

Do not perform unrelated refactors.

Do not silently change existing behavior.

Do not assume ambiguous requirements.

If an architectural requirement is unclear, ask before implementing it.

TESTING

Every meaningful change must include/update tests.

Before a commit run:

    go test ./...

For build-affecting changes run:

    go build -o bin/unishell ./cmd/unishell

Do not instruct the user to commit if tests fail.

GIT WORKFLOW

Use conventional commits.

Every meaningful architectural step must have its own commit.

Examples:

    feat:
    fix:
    refactor:
    test:
    chore:
    ci:
    build:
    security:
    legal:

Explicitly tell the user when a commit should be made and provide the
exact commit command and message.

SECURITY

Do not claim that native binaries cannot be reverse engineered.

Do not claim that cleanup is guaranteed after every possible system
failure.

Do not place secrets in source code, tests, CI configuration, README,
logs, command-line arguments, or Git history.

Do not expose decrypted payloads unnecessarily.

When security-sensitive behavior changes, add negative tests.

AI WORKFLOW

For each requested change:

    1. Inspect the current implementation and tests.
    2. Explain the intended change concisely.
    3. Identify affected files.
    4. Implement the smallest coherent change.
    5. Add/update tests.
    6. Run gofmt.
    7. Run go test ./...
    8. Build when appropriate.
    9. Review the diff.
    10. Tell the user exactly what to commit.
    11. Wait for confirmation before moving to the next architectural step.

Do not rewrite the entire project for a small feature.

Do not recreate the legacy Bash project.

Do not invent requirements.

Keep the ultimate single-binary deployment goal in mind for every
architectural decision.
```

