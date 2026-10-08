" vim: set ft=vim:
" ╔═══════════════════════════════════════════════════════════════
" ║ STYLING THEME AND CUSTOM COLORS

if has('termguicolors')
    " ╭────────────────────────────────╮
    " │ THEME                          │
    " ╰────────────────────────────────╯
    if !empty(globpath(&runtimepath, 'colors/slate.vim'))
        colorscheme slate
    endif

    " ╭────────────────────────────────╮
    " │ COMPONENTS CUSTOM COLORING     │
    " ╰────────────────────────────────╯
    highlight Normal ctermbg=NONE guibg=NONE                       " Remove theme background for tmux dimming to work
    highlight Visual guifg=#d8dee9 guibg=#403820                   " Visual mode highlight colors
    highlight CursorLine guibg=#21252d gui=NONE                    " Cursor line background color
    highlight LineNr guifg=#f0b84b                                 " Line number style
    highlight CursorLineNr guifg=#ffffff guibg=#4b1922 gui=bold    " Current line number style
    " Lighten the comments color more, this is bevtter when SLATE is the
    " colorscheme and the background color is removed like this setup
    highlight Comment guifg=#7f8490
    highlight StatusLine guifg=#ffffff guibg=#16432e           " Active Status Bar colors
    highlight StatusLineNC guifg=#ffffff guibg=#49211f         " Inactive Status Bar colors

    " SEARCH HIGHLIGHTING
    highlight Search    guifg=#d8dee9 guibg=#5a4a2a
    highlight IncSearch guifg=#1f232a guibg=#f0b84b gui=bold

    " ModeMsg
    highlight ModeMsg guifg=#f6c945 guibg=NONE gui=bold

endif
" ╚═══════════════════════════════════════════════════════════════
