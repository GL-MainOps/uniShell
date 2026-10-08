 " vim: set ft=vim:
" ╭────────────────────────────────╮
" │ DIFF RECENT EDITS              │
" ╰────────────────────────────────╯
" Undo diff mode on the real window when the scratch buffer goes away.
function! s:DiffOrigOff(winid) abort
    if exists('*win_execute') && win_id2win(a:winid) > 0
        call win_execute(a:winid, 'diffoff | unlet! w:diff_orig')
    endif
endfunction

function! s:DiffOrig() abort
    " Second call = toggle off (also cleans up if the scratch was closed by hand).
    if exists('w:diff_orig')
        let l:nr = w:diff_orig
        unlet w:diff_orig
        diffoff
        silent! execute 'bwipeout!' l:nr
        return
    endif

    let l:file = expand('%:p')
    if &buftype !=# '' || empty(l:file) || !filereadable(l:file)
        echohl WarningMsg
        echomsg 'DiffOrig: this buffer has no file on disk to compare with'
        echohl None
        return
    endif

    let l:ft   = &filetype
    let l:orig = win_getid()

    " Split BEFORE any diffthis so the new window doesn't inherit diff options.
    " leftabove: saved version on the left, your edits on the right.
    leftabove vnew
    setlocal buftype=nofile bufhidden=wipe nobuflisted noswapfile
    execute 'silent read ++edit' fnameescape(l:file)
    silent 1delete _
    let &l:syntax = l:ft           " highlighting without firing FileType autocmds
    setlocal readonly nomodifiable
    diffthis
    let l:scratch = bufnr('%')
    execute 'autocmd BufWipeout <buffer> call s:DiffOrigOff(' . l:orig . ')'

    call win_gotoid(l:orig)        " back to your real buffer
    diffthis
    let w:diff_orig = l:scratch
endfunction
" @cheat-group FUNCTIONS
" @cheat <leader>do          :DiffOrig<CR>              | DIFF SAVED FILE WITH CURRENT EDITS (TOGGLE)
command! DiffOrig call s:DiffOrig()
nnoremap <silent> <leader>do :DiffOrig<CR>

