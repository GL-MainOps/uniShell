" vim: set ft=vim:
" ╔═══════════════════════════════════════════════════════════════
" ║ CORE VIM SETTINGS

" ╭────────────────────────────────╮
" │ BEHAVIOR                       │
" ╰────────────────────────────────╯
filetype plugin indent on               " Detect file types, load filetype plugins, and enable filetype indentation.
set mouse=a                             " Enable mouse usage (all modes)
set history=1000                        " Keep 1000 lines of command line history
set nospell                             " Disable spellcheck
set notitle                             " Disable setting  window title
set report=0                            " Print how many lines were changed
set autoread                            " Reload files changed externally
set updatetime=1000                     " Wait time befor idle-dependent events start
set encoding=utf-8                      " Default encoding | over SSH the locale may be C/POSIX
scriptencoding utf-8                    " needed because of the non-ASCII listchars below
set nrformats-=octal                    " Ctrl-A on 007 or 010 stays decimal

silent! packadd! matchit                " smarter %: if/fi, <tag></tag>, etc.

if &shell =~# 'fish$'                   " Make sure VIM's shell is posix when using fish
  set shell=/bin/sh
endif

" ┌────────────────┐
" │ TIMEOUTS       │
" └────────────────┘
set ttimeout                            " Time out for key codes
set ttimeoutlen=50                     " Wait up to 50ms after Esc for special key

" ┌────────────────┐
" │ JUMP TO LAST   │
" │ CURSOR POSITION│
" │ ON LAUNCH      │
" └────────────────┘
augroup UserLastPosition
    autocmd!
    autocmd BufReadPost *
        \ if line("'\"") >= 1 && line("'\"") <= line('$')
        \   && index(['gitcommit', 'gitrebase', 'xxd', 'svn', 'hgcommit'], &filetype) == -1
        \ |     execute 'normal! g`"'
        \ | endif
augroup END


" ╭────────────────────────────────╮
" │ DISPLAY                        │
" ╰────────────────────────────────╯
syntax on                               " Enable syntax highlighting
set signcolumn=auto
set scrolloff=0                         " Disable Showing a few lines of context around the cursor.
set novisualbell                        " Disable visual bell
set background=dark                     " Colorscheme background hint
if has('termguicolors')
    set termguicolors                   " Enable 24bit color support in vim
endif

" ┌────────────────┐
" │ WINDOW         │
" │ MANAGEMENT     │
" └────────────────┘
set splitbelow                          " New horizontal splits go below
set splitright                          " New vertical splits go right

" ┌────────────────┐
" │ LINE NUMBERS   │
" └────────────────┘
set number                              " Enable line numbers
set relativenumber                      " Enable line numbers relative to cursor position

" ┌────────────────┐
" │ WRAPPING       │
" └────────────────┘
set wrap                                " Wrap long lines
set linebreak                           " Wrap at word boundaries
set breakindent                         " Indent wrapped lines
set display=truncate                    " Show @@@ in the last line if it is truncated.

" ┌────────────────┐
" │ HIGHLIGHT      │
" │ CURSOR LOCATION│
" └────────────────┘
set nocursorcolumn                      " Don't highlight cursol column
set cursorline                          " Highlight cursol line (Replaced with the augroup below)
" set cursorlineopt=number              " Restrict cursorline highlighting to the line number part

augroup cursorline                      " Enable cursorline only in active windows
    autocmd!
    autocmd WinEnter * setlocal cursorline
    autocmd WinLeave * setlocal nocursorline
augroup END

" ┌────────────────┐
" │ WHITESPACE     │
" │ CHARACTERS     │
" └────────────────┘
set list
set listchars=tab:»·,trail:·,nbsp:␣,extends:›,precedes:‹


" ╭────────────────────────────────╮
" │ TEXT HANDLING                  │
" ╰────────────────────────────────╯
set backspace=indent,eol,start          " Allow backspacing over everything in insert mode.
set nojoinspaces                        " Single space after '.,?,!'


" ┌────────────────┐
" │ FOLDING        │
" └────────────────┘
set foldenable                          " Ensure folding is enabled
set foldlevel=99                        " Start with all folds open
set foldcolumn=2                        " Display fold markers in a dedicated column in the left
set foldnestmax=20                      " Max Depth of folding
set foldminlines=3                      " Minimum lines to allow folding



" ┌────────────────┐
" │ INDENTATION    │
" └────────────────┘
set autoindent                          " Enable Auto Indentation | Maintain the current indentation level
set smarttab                            " Use shiftwidth intelligently at line start
set expandtab                           " Insert spaces instead of tabs
set tabstop=4                           " Display tabs as 4 spaces
set shiftwidth=0                        " Use tabstop for indentation width
set softtabstop=-1                      " Use shiftwidth for Tab/Backspace
set shiftround

" ┌────────────────┐
" │ FILE-SPECIFIC  │
" │ SETTINGS       │
" └────────────────┘


" Fenced code blocks in Markdown get real highlighting; 'bash=sh' aliases bash to sh syntax
let g:markdown_fenced_languages = ['yaml', 'json', 'sh', 'bash=sh', 'groovy', 'dockerfile', 'toml']
let g:markdown_folding = 1          " built-in folding by heading
augroup filetype_settings
    autocmd!
    " formatoptions-=cro => prevents automatic "c: comments formatting" | "r:
    " comment continuation when pressing Enter" | "o: comment continuation with
    " o/O"
    autocmd FileType * setlocal formatoptions-=cro formatoptions+=j

    " YAML
    autocmd FileType yaml setlocal cursorcolumn tabstop=2 softtabstop=2 shiftwidth=2 textwidth=0 foldmethod=indent foldnestmax=3 foldminlines=3 expandtab indentkeys-=0# foldlevel=99

    " JSON
    autocmd FileType json,jsonc setlocal expandtab shiftwidth=2 conceallevel=0 foldmethod=indent foldlevel=99
    autocmd FileType json if executable('jq') | setlocal equalprg=jq\ . | endif

    " Jenkinsfiles
    autocmd BufNewFile,BufRead Jenkinsfile*,*jenkinsfile,JenkinsFile*,*JenkinsFile setfiletype groovy
    autocmd FileType groovy setlocal expandtab shiftwidth=4 commentstring=//\ %s foldmethod=indent foldlevel=99

    " MARKDOWN
    autocmd FileType markdown setlocal tabstop=2 softtabstop=2 shiftwidth=2 textwidth=0 expandtab wrap linebreak foldlevel=99
    autocmd FileType markdown silent! setlocal breakindent
    autocmd FileType markdown nnoremap <buffer> <leader>sp :setlocal spell!<CR>
augroup END

 



" ╭────────────────────────────────╮
" │ SEARCHING                      │
" ╰────────────────────────────────╯
" Do incremental searching when it's possible to timeout.
if has('reltime')
  set incsearch          " Show matches while typing
endif
set ignorecase           " Case-insensitive search
set smartcase            " Case-sensitive if uppercase present
set magic                " Use regex 'magic' mode
set hlsearch             " Keep matches highlighted

" Clear old search highlighting when opening a file
augroup UserClearSearch
    autocmd!
    autocmd BufReadPost,BufNewFile * let @/ = ''
augroup END

" ╚═══════════════════════════════════════════════════════════════
