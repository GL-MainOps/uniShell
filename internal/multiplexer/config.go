package multiplexer

import (
	"os"
)

const (
	DefaultName = "tmux"

	EnvMultiplexer        = "UNISHELL_MULTIPLEXER"
	EnvSession            = "UNISHELL_SESSION"
	EnvMultiplexerSession = "UNISHELL_MULTIPLEXER_SESSION"
)

type Config struct {
	Name            string
	Session         string
	MultiplexerName string
}

func ResolveConfig(
	multiplexerName,
	sessionName,
	multiplexerSessionName string,
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

	return Config{
		Name:            multiplexerName,
		Session:         sessionName,
		MultiplexerName: multiplexerSessionName,
	}
}
