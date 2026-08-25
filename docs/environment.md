# uniShell Environment Variables

uniShell supports environment-variable equivalents for its command-line
configuration.

Command-line flags take precedence over environment variables.

## Multiplexer

### `UNISHELL_MULTIPLEXER`

Selects the multiplexer backend.

Supported values:

- `tmux`
- `zellij`

Default:

```text
tmux
```

Equivalent flag:

```text
--multiplexer
```

### `UNISHELL_SESSION`

Selects the uniShell logical session name.

Default:

```text
default
```

Equivalent flag:

```text
--session
```

This is uniShell's session identity and is independent from the  
multiplexer's native session naming.

### `UNISHELL_MULTIPLEXER_SESSION`

Optionally specifies the native session name passed to the selected  
multiplexer.

Equivalent flag:

```text
--multiplexer-session
```

If unset, uniShell does not override the multiplexer-native session  
naming behavior.

## Configuration precedence

For every configurable option:

```text
CLI flag
    ↓
environment variable
    ↓
built-in default
```

Environment variables are intended to make non-interactive automation  
possible without requiring command-line arguments.
