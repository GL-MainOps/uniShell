 " vim: set ft=vim:
" ╭────────────────────────────────╮
" │ UNIVERSAL COMMENT TOGGLE       │
" ╰────────────────────────────────╯
" filetype -> [left, right, block]
"   right == ''  : line comment
"   block == 1   : wrap the whole selection in ONE left/right pair
"   block == 0   : comment every non-blank line separately
let s:delims = {}

function! s:Reg(fts, left, right, ...) abort
    let l:block = a:0 ? a:1 : 0
    for l:ft in a:fts
        let s:delims[l:ft] = [a:left, a:right, l:block]
    endfor
endfunction

call s:Reg(['yaml', 'toml', 'dockerfile', 'sh', 'bash', 'zsh', 'fish', 'conf',
    \ 'gitconfig', 'terraform', 'hcl', 'make', 'python', 'ruby', 'perl',
    \ 'nginx', 'sshconfig', 'sshdconfig', 'systemd', 'crontab', 'tmux',
    \ 'requirements', 'ps1', 'cmake', 'nix'], '#', '')
call s:Reg(['c', 'cpp', 'cs', 'java', 'javascript', 'javascriptreact',
    \ 'typescript', 'typescriptreact', 'go', 'rust', 'kotlin', 'scala',
    \ 'swift', 'php', 'groovy', 'jenkinsfile', 'gradle', 'json', 'jsonc',
    \ 'json5', 'jsonnet', 'proto', 'cue', 'scss', 'kdl'], '//', '')
call s:Reg(['dosini', 'ini'], ';', '')
call s:Reg(['sql', 'lua'], '--', '')
call s:Reg(['vim'], '"', '')
call s:Reg(['markdown', 'html', 'xml'], '<!--', '-->', 1)
call s:Reg(['css'], '/*', '*/')
call s:Reg(['helm'], '{{/*', '*/}}')

function! s:Delims() abort
    " 'yaml.ansible' -> try 'yaml', then 'ansible'
    for l:ft in split(tolower(&filetype), '\.')
        " Vim9 script uses '#', legacy Vim script uses '"'
        if l:ft ==# 'vim' && match(getline(1, 50), '^\s*vim9script\>') >= 0
            return ['#', '', 0]
        endif
        if has_key(s:delims, l:ft)
            return s:delims[l:ft]
        endif
    endfor
    if &commentstring =~# '%s'
        let l:p = split(&commentstring, '%s', 1)
        return [trim(l:p[0]), trim(get(l:p, 1, '')), 0]
    endif
    return ['', '', 0]
endfunction

function! s:IsCommented(line, left, right) abort
    let l:s = substitute(a:line, '^\s*', '', '')
    let l:s = substitute(l:s, '\s*$', '', '')
    if stridx(l:s, a:left) != 0
        return 0
    endif
    if empty(a:right)
        return 1
    endif
    let l:n = strlen(l:s) - strlen(a:right)
    return l:n >= strlen(a:left) && strpart(l:s, l:n) ==# a:right
endfunction

function! s:Uncomment(line, left, right) abort
    let l:ind  = matchstr(a:line, '^\s*')
    let l:body = strpart(a:line, strlen(l:ind) + strlen(a:left))
    if l:body =~# '^ '                         " one optional space
        let l:body = strpart(l:body, 1)
    endif
    if !empty(a:right)
        let l:body = substitute(l:body,
            \ '\C\V\s\?' . escape(a:right, '\') . '\s\*\$', '', '')
    endif
    return l:ind . l:body
endfunction

" One left/right pair around the whole range (markdown, html, xml).
function! s:ToggleBlock(first, last, left, right) abort
    let l:lines = getline(a:first, a:last)
    let l:fi = match(l:lines, '\S')            " first non-blank line
    if l:fi < 0
        return
    endif
    let l:li = len(l:lines) - 1                " last non-blank line
    while l:lines[l:li] !~# '\S'
        let l:li -= 1
    endwhile

    let l:text = trim(join(l:lines[l:fi : l:li], "\n"))
    let l:ll = strlen(a:left)
    let l:rl = strlen(a:right)

    " Wrapped = ONE comment spanning the selection: opener first, closer last,
    " and no other opener/closer in between (so two adjacent comments don't count).
    let l:wrapped = strlen(l:text) >= l:ll + l:rl
        \ && stridx(l:text, a:left) == 0
        \ && strridx(l:text, a:left) == 0
        \ && stridx(l:text, a:right) == strlen(l:text) - l:rl

    if l:wrapped
        let l:ind = matchstr(l:lines[l:fi], '^\s*')
        let l:lines[l:fi] = l:ind . substitute(
            \ strpart(l:lines[l:fi], strlen(l:ind) + l:ll), '^ ', '', '')
        let l:lines[l:li] = substitute(l:lines[l:li],
            \ '\C\V\s\?' . escape(a:right, '\') . '\s\*\$', '', '')
        " Delimiter-only lines (comment written on its own lines) disappear.
        let l:keep = []
        for l:i in range(len(l:lines))
            if (l:i == l:fi || l:i == l:li) && l:lines[l:i] !~# '\S'
                continue
            endif
            call add(l:keep, l:lines[l:i])
        endfor
    else
        if stridx(l:text, a:right) >= 0
            echohl WarningMsg
            echomsg 'ToggleCommentLines: selection already contains "'
                \ . a:right . '"; nested comments are not supported'
            echohl None
            return
        endif
        let l:ind = matchstr(l:lines[l:fi], '^\s*')
        let l:lines[l:fi] = l:ind . a:left . ' '
            \ . strpart(l:lines[l:fi], strlen(l:ind))
        let l:lines[l:li] = substitute(l:lines[l:li], '\s*$', '', '')
            \ . ' ' . a:right
        let l:keep = l:lines
    endif

    if !empty(l:keep)
        call setline(a:first, l:keep)
    endif
    if len(l:keep) < a:last - a:first + 1
        silent execute (a:first + len(l:keep)) . ',' . a:last . 'delete _'
    endif
endfunction

function! ToggleCommentLines() range abort
    let [l:left, l:right, l:block] = s:Delims()
    if empty(l:left)
        echohl WarningMsg
        echomsg 'ToggleCommentLines: no comment syntax for filetype "' . &filetype . '"'
        echohl None
        return
    endif

    if l:block
        call s:ToggleBlock(a:firstline, a:lastline, l:left, l:right)
        return
    endif

    let l:lines = getline(a:firstline, a:lastline)
    let l:all  = 1     " every non-blank line already commented?
    let l:lead = ''    " indent of the least-indented non-blank line
    let l:min  = -1

    for l:ln in l:lines
        if l:ln !~# '\S'
            continue
        endif
        if !s:IsCommented(l:ln, l:left, l:right)
            let l:all = 0
        endif
        let l:ind = matchstr(l:ln, '^\s*')
        let l:w = strdisplaywidth(l:ind)
        if l:min < 0 || l:w < l:min
            let l:min = l:w
            let l:lead = l:ind
        endif
    endfor
    if l:min < 0
        return         " selection is all blank
    endif

    for l:i in range(len(l:lines))
        let l:ln = l:lines[l:i]
        if l:ln !~# '\S'
            continue
        endif
        if l:all
            let l:lines[l:i] = s:Uncomment(l:ln, l:left, l:right)
        else
            let l:cut = stridx(l:ln, l:lead) == 0
                \ ? strlen(l:lead) : strlen(matchstr(l:ln, '^\s*'))
            let l:lines[l:i] = strpart(l:ln, 0, l:cut) . l:left . ' '
                \ . strpart(l:ln, l:cut)
                \ . (empty(l:right) ? '' : ' ' . l:right)
        endif
    endfor

    call setline(a:firstline, l:lines)
endfunction

" No <C-U>: let Vim insert '<,'> so the function sees the whole selection.
nnoremap <silent> <leader>/ :call ToggleCommentLines()<CR>
xnoremap <silent> <leader>/ :call ToggleCommentLines()<CR>
nnoremap <silent> <C-_> :call ToggleCommentLines()<CR>
xnoremap <silent> <C-_> :call ToggleCommentLines()<CR>
inoremap <silent> <C-_> <C-o>:call ToggleCommentLines()<CR>

