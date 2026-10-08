" vim: set ft=vim:
" ╔═══════════════════════════════════════════════════════════════
" ║ STATUS BAR CONFIGURATIONS

set laststatus=2         " Always show status line
set ruler               " show the cursor position all the time


" ╭──────────────────────────────────╮
" │ COMMANDLINE DISPLAY & COMPLETION │
" ╰──────────────────────────────────╯
set showcmd             " display incomplete commands
set wildmenu            " display completion matches in a status line
set wildmode=longest:full,full
set wildignorecase
if has('patch-8.2.4325')
    set wildoptions=pum             " vertical popup menu instead of one line
endif


" ╭──────────────────────────────────╮
" │ STATUS BAR COMPONENTS            │
" ╰──────────────────────────────────╯
" ┌────────────────┐
" │ FILETYPE       │
" └────────────────┘
function! StatusFileType() abort
    let l:ft = &filetype
    let l:ft = empty(l:ft) ? ' Unknown Filetype' : toupper(l:ft)
    return '[ ' . toupper(&fileformat) . ' | ' . l:ft . ' ]'
endfunction

" ┌────────────────┐
" │ ENCODING       │
" └────────────────┘
function! StatusFileFlags() abort
    let l:f = []
    if !empty(&fileencoding) && &fileencoding !=# 'utf-8'
        call add(l:f, &fileencoding)
    endif
    return empty(l:f) ? '' : '[' . join(l:f, ',') . '] '
endfunction


" ╭──────────────────────────────────╮
" │ STATUS BAR                       │
" ╰──────────────────────────────────╯
let &statusline = join([
            \ '%F %h%m%r%w',
            \ '%=',
            \ '%{&expandtab ? "sp" : "tab"}:%{shiftwidth()}',
            \ '%{StatusFileFlags()}  ',
            \ '%v,%l:%L  %P ',
            \ '%{StatusFileType()}  ',
            \ ], '')

" ╚═══════════════════════════════════════════════════════════════
