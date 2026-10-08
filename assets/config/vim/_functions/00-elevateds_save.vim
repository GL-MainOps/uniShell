 " vim: set ft=vim:
" ╭────────────────────────────────╮
" │ ELEVATED SAVE                  │
" ╰────────────────────────────────╯
function! s:SudoWrite() abort
    let l:file = expand('%:p')
    if &buftype !=# '' || empty(l:file)
        echohl WarningMsg
        echomsg 'SudoWrite: this buffer has no file name'
        echohl None
        return
    endif

    " No sudo needed? (existing file you can write, or new file in a dir you own)
    let l:writable = empty(getftype(l:file))
        \ ? filewritable(fnamemodify(l:file, ':h')) == 2
        \ : filewritable(l:file) == 1
    if l:writable
        write
        return
    endif

    execute 'write !sudo tee ' . shellescape(l:file, 1) . ' > /dev/null'
    if v:shell_error
        echohl ErrorMsg
        echomsg 'SudoWrite: failed (exit ' . v:shell_error . '), buffer left untouched'
        echohl None
        return
    endif

    " ':write !cmd' doesn't clear 'modified' or refresh Vim's idea of the file's
    " mtime, so reload to resync (avoids a later "file changed" warning).
    edit!
endfunction

" @cheat-group FUNCTIONS
" @cheat <leader>W           :SudoWrite<CR>             | SAVE FILE WITHOUT LEAVING IF IT NEEDS ELEVATION
command! SudoWrite call s:SudoWrite()
nnoremap <silent> <leader>W :SudoWrite<CR>

