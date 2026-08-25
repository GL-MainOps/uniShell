package multiplexer

import "os"

const (
	DefaultName = "tmux"

	EnvMultiplexer        = "UNISHELL_MULTIPLEXER"
	EnvSession            = "UNISHELL_SESSION"
	EnvMultiplexerSession = "UNISHELL_MULTIPLEXER_SESSION"
	EnvTmuxSocket         = "UNISHELL_TMUX_SOCKET"
)

type Config struct {
	Name            string
	Session         string
	MultiplexerName string
	TmuxSocket      string
}

func ResolveConfig(
	multiplexerName,
	sessionName,
	multiplexerSessionName,
	tmuxSocket string,
) Config {
	if multiplexerName == "" {
		multiplexerName = os.Getenv(EnvMultiplexer)
	}

	if multiplexerName == "" {
		multiplexerName = DefaultName
	}

	if sessionName == "" {
		sessionName = os.Getenv(EnvSession)
	}

	if sessionName == "" {
		sessionName = "default"
	}

	if multiplexerSessionName == "" {
		multiplexerSessionName = os.Getenv(EnvMultiplexerSession)
	}

	if tmuxSocket == "" {
		tmuxSocket = os.Getenv(EnvTmuxSocket)
	}

	return Config{
		Name:            multiplexerName,
		Session:         sessionName,
		MultiplexerName: multiplexerSessionName,
		TmuxSocket:      tmuxSocket,
	}
}
