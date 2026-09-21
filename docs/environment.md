# uniShell Environment Variables

uniShell supports environment-variable equivalents for its command-line
configuration.

Command-line flags take precedence over environment variables.

Some environment variables are configuration inputs, while others are
provided by uniShell to describe the active runtime session.

## Global Configuration

### `UNISHELL_AUTH_TOKEN`

Provides the authentication token used when accessing the encrypted
runtime bundle.

This variable is primarily relevant to bundle generation and runtime
bundle handling.

The value must not be committed to the repository or embedded directly
in project configuration.

### `UNISHELL_MULTIPLEXER`

Selects the multiplexer behavior for the shell session.

Supported values:

- `tmux`
- `zellij`
- `none`
- `disabled`

Behavior:

- `tmux` launches or reattaches to a tmux session.
- `zellij` launches or reattaches to a Zellij session.
- `none` starts a normal enhanced shell without a multiplexer.
- `disabled` starts a normal enhanced shell without a multiplexer.
- If the variable is unset, uniShell starts a normal enhanced shell
  without a multiplexer.
- Any other value is rejected and, when running interactively,
  uniShell prompts the user to choose between tmux, Zellij, no
  multiplexer, or quitting.

Equivalent flag:

```text
--multiplexer
```

Command-line precedence:

```text
--multiplexer
    ↓
UNISHELL_MULTIPLEXER
    ↓
no multiplexer
```

Examples:

```bash
UNISHELL_MULTIPLEXER=tmux unishell
UNISHELL_MULTIPLEXER=zellij unishell
UNISHELL_MULTIPLEXER=none unishell
UNISHELL_MULTIPLEXER=disabled unishell
unishell --multiplexer tmux
unishell --multiplexer zellij
unishell --multiplexer none
```

When an invalid value is supplied interactively, uniShell presents:

```text
1. tmux
2. zellij
3. none
4. quit
```

Pressing `Ctrl+C` or selecting `quit` cancels startup safely.

### `UNISHELL_SESSION`

Selects the base name of the uniShell logical session.

If neither `UNISHELL_SESSION` nor `--session` is specified, uniShell
generates the session name as:

```text
unnamed@<first-7-characters-of-runtime-session-id>
```

When a name is specified, uniShell generates the session name as:

```text
<specified-name>@<first-7-characters-of-runtime-session-id>
```

The runtime session ID is generated once for the session and is also the
identifier used by the runtime session directory and session metadata.

Examples:

```text
unnamed@81cac8c
development@81cac8c
```

This is uniShell's logical session identity and is independent from the
multiplexer's native session naming.

Equivalent flag:

```text
--session
```

This is uniShell's session identity and is independent from the  
multiplexer's native session naming.

### `UNISHELL_MULTIPLEXER_SESSION`

Optionally specifies the base name of the managed uniShell multiplexer
session.

Equivalent flag:

```text
--multiplexer-session
```

When specified, uniShell derives the managed multiplexer session name as:

```text
<specified-name>@<first-3-characters-of-runtime-session-id>
```

When unset or blank, uniShell uses:

```text
uS@<first-3-characters-of-runtime-session-id>
```

The runtime session ID is generated once for the session and is the same
identifier used by the runtime session directory and session metadata.

Examples:

```text
work@abc
development@81c
uS@7f2
```

The managed multiplexer session name is used by uniShell when discovering
and reattaching to an existing multiplexer session.

The selected multiplexer backend may use a separate native session name.
The managed uniShell multiplexer session name does not require the backend
to use the same native name.

### Logical and Multiplexer Session Names

uniShell maintains two distinct session-name concepts for multiplexer
sessions:

- `UNISHELL_SESSION` identifies the uniShell logical session.
- `UNISHELL_MULTIPLEXER_SESSION` selects the base name used to derive the
    managed uniShell multiplexer session name.

The logical session name is part of uniShell session metadata and remains
independent of the multiplexer implementation.

The managed multiplexer session name is canonicalized by uniShell and
includes a short runtime-session identifier suffix.

The backend-native session name remains implementation-specific and is
managed by the selected multiplexer backend.

For multiplexer sessions, the managed uniShell multiplexer session name
is also persisted in the session's `.session.json` metadata as:

```json
"multiplexer_session_name": "<managed-multiplexer-session-name>"
```

This field contains the canonical managed uniShell multiplexer session
name, such as `uS@abc` or `work@abc`. It does not contain the backend-native
session name.

The `multiplexer_session_name` field is present only for multiplexer
sessions. Direct-shell session metadata does not contain this field.

## Runtime

### `UNISHELL_RUNTIME_DIR`

Overrides the root directory used by uniShell for its runtime data.

Default:

```text
/var/tmp/.lesscache
```

Resolution precedence:

```text
explicit runtime root
    ↓
UNISHELL_RUNTIME_DIR
    ↓
/var/tmp/.lesscache
```

This variable is an **input** to runtime-root selection.

It does not identify the specific runtime session currently being  
used.

### `UNISHELL_SESSION_RUNTIME_DIR`

Contains the absolute path of the runtime directory belonging to the  
currently active uniShell session.

Example:

```text
/var/tmp/.lesscache/runtime/development/80790914fd006a7e29c4d6945463b371
```

This variable is provided by uniShell to the environment inherited by the
shell and, when applicable, the multiplexer and processes it launches.

It can be used by tools and configuration files that need to locate  
files belonging to the active uniShell runtime.

For example:

```bash
"$UNISHELL_SESSION_RUNTIME_DIR/config/vim/vimrc"
```

Unlike `UNISHELL_RUNTIME_DIR`, this variable is not a user configuration  
input. Its value is determined by uniShell from the active runtime  
session.

When reattaching to an existing uniShell session, it continues to  
refer to the existing session's runtime directory.

### Session Environment

uniShell provides session metadata to the environment inherited by the
direct shell and by processes launched within the managed session.

For multiplexer sessions, these variables are injected before the
multiplexer backend creates the native session, so the multiplexer and
its child shells inherit the same uniShell session environment.

These variables describe the active uniShell session:

### `UNISHELL_SESSION_ID`

Contains the unique runtime session identifier.

This value corresponds to the session identifier stored in the active
session metadata.

### `UNISHELL_SESSION_VERSION`

Contains the uniShell session metadata version.

### `UNISHELL_SESSION_MODE`

Contains the active uniShell session mode.

### `UNISHELL_SESSION_SHELL_NAME`

Contains the logical name of the shell selected for the active session.

### `UNISHELL_SESSION_NAME`

Contains the uniShell logical session name.

This is the logical session identity and is independent from any native
multiplexer session name.

### `UNISHELL_SESSION_SHELL_PROFILE`

Contains the shell profile selected for the active session.

When no shell profile is selected, the value is:

```text
none
```

### `UNISHELL_SESSION_MULTIPLEXER`

When a multiplexer is associated with the active session, contains the
selected multiplexer name.

This variable is omitted when the session does not use a multiplexer.

### `UNISHELL_SESSION_MULTIPLEXER_ENDPOINT`

When a multiplexer is associated with the active session, contains the
canonical endpoint resolved by the selected multiplexer backend.

The endpoint is backend-specific. uniShell does not construct or
interpret the backend endpoint in the application layer.

This variable is omitted when the session does not use a multiplexer.

These variables are **runtime-provided values**, not configuration
inputs. They are derived from the active session metadata.

For a direct shell session, the values are injected into the shell
environment after uniShell resolves the requested shell and shell
profile.

For a multiplexer session, the values are injected before the backend
creates the native multiplexer session, allowing the multiplexer and its
child processes to inherit the same session environment.

The persisted session metadata is the canonical source for these
values; shell configuration files do not need to parse `.session.json`
directly.


## Multiplexer Configuration

### `UNISHELL_TMUX_OPTS`

Provides additional arguments to tmux when uniShell launches it.

The value is parsed as shell-style arguments and is specific to the tmux  
backend.

This allows users to customize tmux invocation without modifying the  
uniShell source or bundled tmux configuration.

### `UNISHELL_ZELLIJ_OPTS`

Provides additional arguments to Zellij when uniShell launches it.

The value is parsed as shell-style arguments and is specific to the  
Zellij backend.

This allows users to customize Zellij invocation without modifying the  
uniShell source or bundled Zellij configuration.

## Configuration Precedence

For configurable options:

```text
CLI flag
    ↓
environment variable
    ↓
built-in default
```

Environment variables are intended to make non-interactive automation  
possible without requiring command-line arguments.

`UNISHELL_RUNTIME_DIR` follows the runtime-root resolution described  
above.

`UNISHELL_SESSION_RUNTIME_DIR` is different: it is generated by  
uniShell and describes the runtime directory of the active session.

## Reattaching

When uniShell creates a new session, `UNISHELL_SESSION_RUNTIME_DIR`  
points to the runtime directory created for that session.

When uniShell reattaches to an existing session, the existing runtime  
directory is reused. A new runtime session directory is not created.

For example:

```text
First invocation:

/var/tmp/.lesscache/runtime/development/<session-id>


Reattach:

/var/tmp/.lesscache/runtime/development/<same-session-id>
```

Therefore, `UNISHELL_SESSION_RUNTIME_DIR` remains stable for the  
lifetime of the managed session.

`UNISHELL_RUNTIME_DIR` may be absent during a later reattach if the user  
does not provide the override again. This does not affect  
`UNISHELL_SESSION_RUNTIME_DIR`, which is recovered from the existing  
session.


## `clean` Command

### `--target <session-name>`

Selects a specific uniShell session for cleanup.

Example:

```bash
unishell clean --target development
```

`--target` is a command-line-only option.

When `--target` is not supplied, `clean` interactively determines which
managed session should be cleaned.

`clean` never deletes a session without explicit user confirmation.

If the specified session does not exist, uniShell reports that the target
was not found and does not delete another session.

For safety, `clean` does not accept positional arguments.

When `--target` is not supplied, uniShell displays a hint showing that a
specific session can be selected directly:

```text
Hint: use --target <session-name> to select a session directly.
Example: unishell clean --target development
```
