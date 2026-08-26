package api

type Options struct {
	Tmux   TmuxOptions
	Zellij ZellijOptions
}

type TmuxOptions struct {
	CreateArgs []string
}

type ZellijOptions struct {
	CreateArgs []string
}
