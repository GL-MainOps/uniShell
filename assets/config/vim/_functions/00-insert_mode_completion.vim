" vim: set ft=vim:
" ╭────────────────────────────────╮
" │ INSERT-MODE COMPLETION         │
" ╰────────────────────────────────╯
" NOTES: remove the auto-popup block once you install an LSP client, since only
" one thing should open the popup
" Complete from this buffer, other windows and loaded buffers only. Drop the
" default 'i' (include files) and 't' (tags) sources: they are slow.
set complete=.,w,b
set completeopt=menu,menuone,noinsert,noselect

" Tab: next match, or complete after a word character, else a real <Tab>.
function! s:Tab() abort
    if pumvisible() || strpart(getline('.'), 0, col('.') - 1) =~# '\k$'
        return "\<C-n>"
    endif
    return "\<Tab>"
endfunction
inoremap <expr> <Tab>   <SID>Tab()
inoremap <expr> <S-Tab> pumvisible() ? "\<C-p>" : "\<S-Tab>"

" ── Auto-popup after N typed characters ──────────────────────────────
let g:autocomplete_minchars = 3          " change this to taste (minimum 2)

set shortmess+=c                         " no "Pattern not found" / "match 1 of 5" messages

function! s:AutoComplete() abort
    if pumvisible() || v:char !~# '\k'
        return
    endif
    let l:line = getline('.')
    let l:col  = col('.') - 1            " bytes before the cursor
    let l:n    = max([g:autocomplete_minchars, 2])

    " The N-1 characters before the cursor are word characters, the first is
    " not a digit and starts the word; the char after the cursor is not a word char.
    let l:pat = '\k\@<!\K\k\{' . (l:n - 2) . '}$'
    if strpart(l:line, 0, l:col) =~# l:pat
        \ && strpart(l:line, l:col) !~# '^\k'
        call feedkeys("\<C-n>", 'n')
    endif
endfunction

augroup AutoComplete
    autocmd!
    autocmd InsertCharPre * call s:AutoComplete()
augroup END

