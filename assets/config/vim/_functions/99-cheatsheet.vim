" vim: set ft=vim:
 " ============================================================================
" cheatsheet.vim -- :Cheat, a scratch-buffer cheatsheet built from the
"                   "@cheat" comments scattered through your own config.
"
" Install:  ~/.vim/plugin/cheatsheet.vim
"     or:   source ~/path/to/cheatsheet.vim   " from your vimrc
"
" Works in Vim 8.x, Vim 9.x and Neovim. Uses ripgrep when present, falls
" back to pure Vimscript when it isn't.
" ----------------------------------------------------------------------------
" COMMENT SYNTAX -- put these anywhere in your vim config
"
"   " @cheat-group Line editing
"   " @cheat <M-j>  <M-Down>  ]e | move line or selection down
"   " @cheat <Leader>d          | duplicate line or selection
"   " @cheat Alt keys need terminal support -- see :h ttimeoutlen
"
"   @cheat-group  sets the section heading for every @cheat below it
"                 IN THE SAME FILE (resets to "General" in the next file)
"   @cheat        keys | description      -> a key row
"   @cheat        free text, no pipe      -> a note under the current group
"   @cheat-ignore anywhere in a file: that whole file contributes nothing
"
" @cheat-ignore  (which is why the examples above are not real entries)
"
" USAGE
"   :Cheat           open the cheatsheet
"   :Cheat move      open it filtered to rows matching "move"
"   :Cheat!          rescan your config and open (cache is also dropped
"                    automatically whenever you :write a .vim file)
"   <Leader>?        same as :Cheat
"
" Inside the window:  q close   / search   y yank the keys on this line
" ============================================================================

if exists('g:loaded_cheatsheet') || &compatible
  finish
endif
let g:loaded_cheatsheet = 1

let s:save_cpo = &cpo
set cpo&vim

" ----------------------------------------------------------------------------
" Options -- set in your vimrc BEFORE this file is sourced
" ----------------------------------------------------------------------------

" Files and directories to scan. Empty = auto-detect $MYVIMRC and ~/.vim.
let g:cheat_paths = get(g:, 'cheat_paths', [])

" Window style: 'vsplit' | 'split' | 'tab' | 'current'
" 'vsplit' falls back to a horizontal split when the window is narrow.
let g:cheat_open = get(g:, 'cheat_open', 'vsplit')

" Width of the key column. 0 = auto-fit to the widest key row.
let g:cheat_key_width = get(g:, 'cheat_key_width', 0)

" Install the <Leader>? mapping.
let g:cheat_default_maps = get(g:, 'cheat_default_maps', 1)

let s:cache  = []
let s:bufnr  = -1
let s:source = ''

" ----------------------------------------------------------------------------
" Where to look
" ----------------------------------------------------------------------------

function! s:Paths() abort
  if !empty(g:cheat_paths)
    return filter(map(copy(g:cheat_paths), 'expand(v:val)'),
          \ 'filereadable(v:val) || isdirectory(v:val)')
  endif

  let l:paths = []
  if !empty($MYVIMRC) && filereadable(expand($MYVIMRC))
    call add(l:paths, expand($MYVIMRC))
  endif
  for l:dir in ['~/.vim', '~/.config/nvim', '~/vimfiles']
    let l:d = expand(l:dir)
    if isdirectory(l:d)
      call add(l:paths, l:d)
    endif
  endfor
  return l:paths
endfunction

" ----------------------------------------------------------------------------
" Collecting raw "file:lnum:text" rows
" ----------------------------------------------------------------------------

function! s:CollectRg(paths) abort
  let l:cmd = 'rg --no-messages --no-heading --with-filename --line-number'
        \ . ' --color never --max-filesize 1M'
        \ . ' --glob "!**/.git/**" --glob "!**/plugged/**"'
        \ . ' --glob "!**/pack/**"  --glob "!**/bundle/**"'
        \ . ' --glob "!**/plug.vim"'
        \ . ' -e "@cheat"'
  for l:p in a:paths
    let l:cmd .= ' ' . shellescape(l:p)
  endfor
  let l:out = systemlist(l:cmd)
  " rg exits 1 for "no matches", which is not an error for us.
  if v:shell_error > 1
    return []
  endif
  return l:out
endfunction

function! s:CollectNative(paths) abort
  let l:files = []
  for l:p in a:paths
    if filereadable(l:p)
      call add(l:files, l:p)
    elseif isdirectory(l:p)
      " Deliberately narrow globs: never walk plugged/ or pack/.
      for l:pat in ['/*.vim', '/vimrc', '/init.vim',
            \       '/plugin/**/*.vim', '/after/**/*.vim',
            \       '/ftplugin/**/*.vim', '/autoload/**/*.vim']
        call extend(l:files, glob(l:p . l:pat, 0, 1))
      endfor
    endif
  endfor

  let l:out = []
  for l:f in l:files
    if !filereadable(l:f)
      continue
    endif
    let l:n = 0
    for l:line in readfile(l:f)
      let l:n += 1
      if stridx(l:line, '@cheat') >= 0
        call add(l:out, l:f . ':' . l:n . ':' . l:line)
      endif
    endfor
  endfor
  return l:out
endfunction

" ----------------------------------------------------------------------------
" Parsing
" ----------------------------------------------------------------------------

function! s:CmpRows(a, b) abort
  if a:a.file ==# a:b.file
    return a:a.lnum - a:b.lnum
  endif
  return a:a.file <# a:b.file ? -1 : 1
endfunction

" Returns [{'name': str, 'items': [{'keys': str, 'desc': str}, ...]}, ...]
function! s:Build() abort
  let l:paths = s:Paths()
  if empty(l:paths)
    let s:source = 'nothing to scan'
    return []
  endif

  if executable('rg')
    let s:source = 'rg'
    let l:raw = s:CollectRg(l:paths)
  else
    let s:source = 'vimscript'
    let l:raw = s:CollectNative(l:paths)
  endif

  " Split, and drop duplicates (a vimrc living inside ~/.vim gets seen twice).
  let l:seen = {}
  let l:rows = []
  for l:r in l:raw
    let l:m = matchlist(l:r, '^\(.\{-}\):\(\d\+\):\(.*\)$')
    if empty(l:m)
      continue
    endif
    let l:key = l:m[1] . ':' . l:m[2]
    if has_key(l:seen, l:key)
      continue
    endif
    let l:seen[l:key] = 1
    call add(l:rows, {'file': l:m[1], 'lnum': str2nr(l:m[2]), 'text': l:m[3]})
  endfor
  call sort(l:rows, function('s:CmpRows'))

  " A file containing @cheat-ignore contributes nothing. Lets a file document
  " the syntax (or show examples) without polluting the cheatsheet.
  let l:ignored = {}
  for l:row in l:rows
    if l:row.text =~# '@cheat-ignore'
      let l:ignored[l:row.file] = 1
    endif
  endfor
  if !empty(l:ignored)
    call filter(l:rows, '!has_key(l:ignored, v:val.file)')
  endif

  let l:groups  = []
  let l:index   = {}
  let l:current = 'General'
  let l:lastfile = ''

  for l:row in l:rows
    if l:row.file !=# l:lastfile
      let l:current  = 'General'
      let l:lastfile = l:row.file
    endif

    " Skip this plugin's own documentation of the syntax.
    if l:row.text =~# '@cheat\%(-group\)\=\s*$'
      continue
    endif

    let l:g = matchlist(l:row.text, '@cheat-group\s\+\(.\{-}\)\s*$')
    if !empty(l:g) && !empty(l:g[1])
      let l:current = l:g[1]
      continue
    endif

    let l:e = matchlist(l:row.text, '@cheat\s\+\(.\{-}\)\s*$')
    if empty(l:e) || empty(l:e[1])
      continue
    endif

    let l:body = l:e[1]
    let l:bar  = stridx(l:body, '|')
    if l:bar >= 0
      let l:item = {'keys': s:Trim(strpart(l:body, 0, l:bar)),
            \       'desc': s:Trim(strpart(l:body, l:bar + 1))}
    else
      let l:item = {'keys': '', 'desc': s:Trim(l:body)}
    endif

    if !has_key(l:index, l:current)
      let l:index[l:current] = len(l:groups)
      call add(l:groups, {'name': l:current, 'items': []})
    endif
    call add(l:groups[l:index[l:current]].items, l:item)
  endfor

  return l:groups
endfunction

function! s:Trim(s) abort
  return substitute(a:s, '^\s*\|\s*$', '', 'g')
endfunction

" ----------------------------------------------------------------------------
" Rendering
" ----------------------------------------------------------------------------

function! s:Matches(group, item, filter) abort
  if empty(a:filter)
    return 1
  endif
  return a:group =~? a:filter || a:item.keys =~? a:filter
        \ || a:item.desc =~? a:filter
endfunction

function! s:Pad(s, width) abort
  let l:gap = a:width - strdisplaywidth(a:s)
  return a:s . (l:gap > 0 ? repeat(' ', l:gap) : '')
endfunction

function! s:Render(groups, filter) abort
  " Auto-fit the key column across everything that survived the filter.
  let l:width = g:cheat_key_width
  if l:width <= 0
    let l:width = 10
    for l:g in a:groups
      for l:it in l:g.items
        if s:Matches(l:g.name, l:it, a:filter)
          let l:width = max([l:width, strdisplaywidth(l:it.keys)])
        endif
      endfor
    endfor
    let l:width = min([l:width, 40])
  endif

  let l:title = empty(a:filter)
        \ ? 'Cheatsheet ~'
        \ : 'Cheatsheet  (filter: ' . a:filter . ') ~'

  let l:lines = [l:title, '']
  let l:total = 0

  for l:g in a:groups
    let l:rows = []
    for l:it in l:g.items
      if !s:Matches(l:g.name, l:it, a:filter)
        continue
      endif
      if empty(l:it.keys)
        call add(l:rows, '    ' . l:it.desc)
      else
        call add(l:rows, '  ' . s:Pad(l:it.keys, l:width) . '  ' . l:it.desc)
      endif
    endfor

    if empty(l:rows)
      continue
    endif
    let l:total += len(l:rows)
    call add(l:lines, l:g.name . ' ~')
    call extend(l:lines, l:rows)
    call add(l:lines, '')
  endfor

  if l:total == 0
    call add(l:lines, '  (nothing matched "' . a:filter . '")')
    call add(l:lines, '')
  endif

  call add(l:lines, repeat('=', 78))
  call add(l:lines, 'q close    / search    y yank keys on this line'
        \ . '    :Cheat! rescan')
  call add(l:lines, l:total . ' entries, scanned via ' . s:source . '.')
  return l:lines
endfunction

" ----------------------------------------------------------------------------
" Window
" ----------------------------------------------------------------------------

function! s:OpenWindow() abort
  if s:bufnr > 0 && bufexists(s:bufnr)
    let l:win = bufwinnr(s:bufnr)
    if l:win > 0
      execute l:win . 'wincmd w'
      setlocal modifiable noreadonly
      silent %delete _
      return
    endif
  endif

  if g:cheat_open ==# 'tab'
    tabnew
  elseif g:cheat_open ==# 'current'
    enew
  elseif g:cheat_open ==# 'split'
    new
  else
    if winwidth(0) >= 120
      vnew
    else
      new
    endif
  endif

  let s:bufnr = bufnr('%')
  silent! file [Cheatsheet]

  setlocal buftype=nofile bufhidden=hide noswapfile nobuflisted
  setlocal nonumber norelativenumber nowrap nolist nospell nofoldenable
  setlocal cursorline

  nnoremap <buffer><silent> q     :close<CR>
  nnoremap <buffer><silent> R     :Cheat!<CR>
  nnoremap <buffer><silent> y     :call <SID>YankKeys()<CR>
  nnoremap <buffer><silent> <CR>  :call <SID>YankKeys()<CR>
endfunction

function! s:YankKeys() abort
  let l:keys = matchstr(getline('.'), '^\s\+\zs\S.\{-}\ze\s\{2,}')
  if empty(l:keys)
    return
  endif
  let @" = l:keys
  if has('clipboard')
    let @+ = l:keys
  endif
  echo 'yanked: ' . l:keys
endfunction

" ----------------------------------------------------------------------------
" Entry point
" ----------------------------------------------------------------------------

function! s:Cheat(refresh, filter) abort
  if a:refresh || empty(s:cache)
    let s:cache = s:Build()
  endif

  if empty(s:cache)
    echohl WarningMsg
    echomsg 'cheatsheet: no @cheat comments found in '
          \ . join(s:Paths(), ', ')
    echohl None
    return
  endif

  let l:lines = s:Render(s:cache, a:filter)
  call s:OpenWindow()
  setlocal modifiable noreadonly
  silent! %delete _
  call setline(1, l:lines)
  " ft=help last: key names in <angle brackets> and the trailing-~ headings
  " get highlighted for free by Vim's own help syntax.
  setlocal filetype=help
  setlocal nomodifiable nomodified readonly
  normal! gg
endfunction

function! s:Complete(arglead, cmdline, cursorpos) abort
  if empty(s:cache)
    let s:cache = s:Build()
  endif
  return filter(map(copy(s:cache), 'v:val.name'),
        \ 'v:val =~? "^" . a:arglead')
endfunction

command! -bang -nargs=? -complete=customlist,<SID>Complete
      \ Cheat call s:Cheat(<bang>0, <q-args>)

if g:cheat_default_maps
  nnoremap <silent> <Leader>? :Cheat<CR>
endif

" Edit your config, get a fresh cheatsheet next time.
augroup cheatsheet_invalidate
  autocmd!
  autocmd BufWritePost *.vim,vimrc,.vimrc,init.vim,.gvimrc let s:cache = []
augroup END

let &cpo = s:save_cpo
unlet s:save_cpo



