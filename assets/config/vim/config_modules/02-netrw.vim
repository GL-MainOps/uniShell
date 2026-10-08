" vim: set ft=vim:
" ╔═══════════════════════════════════════════════════════════════
" ║ NETRW FILE BROWSER CONFIGURATION

let g:netrw_banner=0         " Hide netrw banner
let g:netrw_liststyle=3      " Use tree-style listing
let g:netrw_showhide=1       " Show hidden files by default
let g:netrw_winsize=15       " Sidebar width (%)
" let g:netrw_altv=1           " Open vertical splits on the opposite side
" let g:netrw_browse_split=4   " Open files in the previous window
let g:netrw_preview=1        " Use vertical preview windows
let g:netrw_sizestyle='H'    " Use human-readable file sizes


" ╭────────────────────────────────╮
" │ STARTUP                        │
" ╰────────────────────────────────╯
" Open netrw automatically and return focus to the editing window.
augroup netrw_startup
    autocmd!
    autocmd VimEnter * Lexplore | wincmd p
augroup END


" ╭────────────────────────────────╮
" │ WINDOW BEHAVIOR                │
" ╰────────────────────────────────╯
" Quit Vim instead of leaving netrw as the only window.
augroup netrw_behavior
    autocmd!
    autocmd BufEnter * if &filetype ==# 'netrw' && winnr('$') == 1 | quit | endif
augroup END


" ╭────────────────────────────────╮
" │ NAVIGATION                     │
" ╰────────────────────────────────╯
" Toggle the netrw sidebar.
nnoremap <leader>e :Lexplore<CR>



" Focus the existing netrw window.

" If netrw is not open, open it first.
function! FocusNetrw()
    for l:winnr in range(1, winnr('$'))
        if getwinvar(l:winnr, '&filetype') ==# 'netrw'
            execute l:winnr . 'wincmd w'
            return
        endif
    endfor

    Lexplore
endfunction

nnoremap <leader>n :call FocusNetrw()<CR>

" ╚═══════════════════════════════════════════════════════════════
