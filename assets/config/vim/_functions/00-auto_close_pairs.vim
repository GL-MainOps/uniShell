 " vim: set ft=vim:
" ╭────────────────────────────────╮
" │ AUTO-CLOSING CHARACTERS        │
" ╰────────────────────────────────╯
" Works on Vim 8.x and 9.x (plain Vim script; needs 7.4.849+ for <C-g>U).
"
"   (  [  {  "  '  `   insert the pair, cursor inside
"   )  ]  }  "  '  `   typed in front of the same char: step over it
"   <BS>               inside an empty pair: delete both chars
"   <CR>               between {} [] (): open a block, cursor on middle line
"   /*                 in C-like files: expands to /**/ with cursor inside
"   :AutoPairsToggle   turn everything on/off
"   <C-v>(             always inserts a single literal "("

let g:autopairs_enabled = get(g:, 'autopairs_enabled', 1)

" Default pairs. Per-filetype variants are set in s:FiletypeSetup() below.
let g:autopairs = {
      \ '(': ')', '[': ']', '{': '}',
      \ '"': '"', "'": "'", '`': '`',
      \ }

" Only auto-close when the next char is end-of-line, whitespace or one of these.
" (Typing "(" right before a word will NOT add a ")".)
let s:next_ok = '[[:space:])\]}>,;:.]'

" ── helpers ──────────────────────────────────────────────────────────────────

" [char before cursor, char after cursor] (multibyte safe, '' at line edges)
function! s:Around() abort
  let l:line = getline('.')
  let l:col  = col('.') - 1
  return [matchstr(strpart(l:line, 0, l:col), '.$'),
        \ matchstr(strpart(l:line, l:col), '^.')]
endfunction

function! s:Pairs() abort
  return get(b:, 'autopairs', g:autopairs)
endfunction

" Copy of g:autopairs with entries added/removed
function! s:Variant(add, drop) abort
  let l:pairs = extend(copy(g:autopairs), a:add)
  for l:key in a:drop
    if has_key(l:pairs, l:key)
      call remove(l:pairs, l:key)
    endif
  endfor
  return l:pairs
endfunction

" Python string prefixes: f"..", rb'..', etc.
function! s:IsStringPrefix() abort
  return &filetype ==# 'python'
        \ && strpart(getline('.'), 0, col('.') - 1) =~# '\<[rRbBfFuU]\{1,2}$'
endfunction

" ── <expr> mapping targets ───────────────────────────────────────────────────

" Opening bracket or quote
function! s:Open(char) abort
  let l:pairs = s:Pairs()
  if !g:autopairs_enabled || !has_key(l:pairs, a:char)
    return a:char
  endif
  let [l:prev, l:next] = s:Around()
  let l:close = l:pairs[a:char]

  if a:char ==# l:close                       " quote-like: same open/close char
    if l:next ==# a:char                      " closing quote: step over it
      return "\<C-g>U\<Right>"
    endif
    if l:prev ==# '\' || l:prev ==# a:char    " escaped quote, or """ / ''' run
      return a:char
    endif
    if l:prev =~# '\k' && !s:IsStringPrefix() " don't, it's, closing quote of x"
      return a:char
    endif
  endif

  if l:next !=# '' && l:next !~# s:next_ok    " before a word: leave it alone
    return a:char
  endif
  return a:char . l:close . "\<C-g>U\<Left>"  " <C-g>U keeps undo and . intact
endfunction

" Closing bracket: step over an identical char instead of inserting another
function! s:Close(char) abort
  if g:autopairs_enabled
        \ && s:Around()[1] ==# a:char
        \ && index(values(s:Pairs()), a:char) >= 0
    return "\<C-g>U\<Right>"
  endif
  return a:char
endfunction

" <BS> inside an empty pair removes both halves
function! s:Backspace() abort
  if g:autopairs_enabled
    let [l:prev, l:next] = s:Around()
    let l:pairs = s:Pairs()
    if has_key(l:pairs, l:prev) && l:pairs[l:prev] ==# l:next
      return "\<BS>\<Del>"
    endif
  endif
  return "\<BS>"
endfunction

" <CR> between {} [] () opens an indented block
function! s:Enter() abort
  if g:autopairs_enabled
    let [l:prev, l:next] = s:Around()
    if index(['{}', '[]', '()'], l:prev . l:next) >= 0
      return "\<CR>\<C-o>O"
    endif
  endif
  return "\<CR>"
endfunction

" "/*" -> "/**/" (only in filetypes with C-style comments, and only when the
" "/" starts a token, so globs like src/**/*.js are left alone)
function! s:Star() abort
  if g:autopairs_enabled && &comments =~# '/\*'
    let l:before = strpart(getline('.'), 0, col('.') - 1)
    if l:before =~# '\%(^\|[[:space:]({,;=]\)/$' && s:Around()[1] !~# '\k'
      return "**/\<C-g>U\<Left>\<C-g>U\<Left>"
    endif
  endif
  return '*'
endfunction

" ── mappings ─────────────────────────────────────────────────────────────────
" These use <C-r>= rather than <expr> on purpose: when several keys arrive at
" once (pasting without bracketed paste, macros, laggy SSH, :normal), Vim
" evaluates <expr> mappings *before* the preceding characters are inserted, so
" they see stale text (e.g. "don't" -> "don't'", "(x<BS>" eats the ")").
" <C-r>= is evaluated in order, so the helpers always see the real buffer.

inoremap <silent> (     <C-r>=<SID>Open('(')<CR>
inoremap <silent> [     <C-r>=<SID>Open('[')<CR>
inoremap <silent> {     <C-r>=<SID>Open('{')<CR>
inoremap <silent> "     <C-r>=<SID>Open('"')<CR>
inoremap <silent> '     <C-r>=<SID>Open("'")<CR>
inoremap <silent> `     <C-r>=<SID>Open('`')<CR>
inoremap <silent> <lt>  <C-r>=<SID>Open('<')<CR>
inoremap <silent> )     <C-r>=<SID>Close(')')<CR>
inoremap <silent> ]     <C-r>=<SID>Close(']')<CR>
inoremap <silent> }     <C-r>=<SID>Close('}')<CR>
inoremap <silent> >     <C-r>=<SID>Close('>')<CR>
inoremap <silent> *     <C-r>=<SID>Star()<CR>
inoremap <silent> <CR>  <C-r>=<SID>Enter()<CR>
inoremap <silent> <BS>  <C-r>=<SID>Backspace()<CR>
inoremap <silent> <C-h> <C-r>=<SID>Backspace()<CR>

command! AutoPairsToggle
      \ let g:autopairs_enabled = !g:autopairs_enabled
      \ | echo 'AutoPairs ' . (g:autopairs_enabled ? 'on' : 'off')

" ── per-filetype tweaks ──────────────────────────────────────────────────────

function! s:FiletypeSetup() abort
  unlet! b:autopairs
  let l:ft = &filetype
  if l:ft ==# 'vim'
    let b:autopairs = s:Variant({}, ['"'])            " a double quote starts a comment
  elseif l:ft ==# 'rust'
    let b:autopairs = s:Variant({}, ["'"])            " 'a lifetimes
  elseif l:ft =~# '^\%(lisp\|scheme\|clojure\|racket\)$'
    let b:autopairs = s:Variant({}, ["'", '`'])       " quote operators, not strings
  elseif l:ft =~# '^\%(html\|xml\|xhtml\|htmldjango\)$'
    let b:autopairs = s:Variant({'<': '>'}, [])       " <tag> pairs, only where safe
  endif
endfunction

augroup AutoPairs
  autocmd!
  autocmd FileType * call s:FiletypeSetup()
augroup END
