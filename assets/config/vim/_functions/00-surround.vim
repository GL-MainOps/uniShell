" vim: set ft=vim:
" ============================================================================
" surround-lite.vim -- add, change and delete surroundings. That's all.
"
" A replacement for tpope/vim-surround that keeps the three commands people
" actually use and drops tags, LaTeX, function calls, Lisp, the insert-mode
" mappings and the custom-definition machinery.
"
" The key bindings are the same as vim-surround's, so muscle memory carries
" over. It leans on Vim's own text objects to find pairs instead of
" hand-rolled searching, which is why it fits in a couple hundred lines.
" ----------------------------------------------------------------------------
" @cheat-group Surround
" @cheat {char} below is ONE delimiter character -- see the wrap table.
" @cheat {motion} is any vim motion or text object: iw (inner word), i"
" @cheat (inside quotes), ap (a paragraph), $ (to end of line), t, (up to
" @cheat the next comma). Only sa takes one; the rest know their target.
" @cheat sa{motion}{char} | ADD     saiw)  turns word into (word)
" @cheat sd{char}         | DELETE  sd)    turns (word) into word
" @cheat sc{char}{char}   | CHANGE  sc)]   turns (word) into [word]
" @cheat ss{char}         | LINE    ss)    wraps the whole line
" @cheat s{char}  visual  | wraps the selection
" @cheat Mnemonic: s = surround, then the vim verb -- a/d/c, and ss like dd.
" @cheat ) b  } B  ] r  > a | wrap tight: (x) {x} [x] <x>
" @cheat ( { [ <       | wrap roomy: ( x ) { x } [ x ] < x >
" @cheat " ' `         | wrap in that quote
" @cheat {punctuation} | any other punctuation wraps with itself: *x*
" @cheat 2sd(  sd2(    | a count reaches outward to the 2nd enclosing pair
" @cheat As a target the opening bracket also strips the inner spaces:
" @cheat sd( turns ( x ) into x, while sd) leaves " x " alone.
" @cheat Selecting whole lines (V) then s puts the delimiters on their own
" @cheat lines; a charwise selection wraps inline.
" @cheat Every command prompts in the message line, so a pause means it is
" @cheat waiting for you, not broken. <Esc> cancels.
" ============================================================================

if exists('g:loaded_surround_lite') || &compatible
  finish
endif
let g:loaded_surround_lite = 1

let s:save_cpo = &cpo
set cpo&vim

let g:surround_lite_no_mappings = get(g:, 'surround_lite_no_mappings', 0)

" The prefix every command hangs off. Default "s" shadows normal-mode s
" (substitute one character) -- cl does exactly the same thing. If you use
" s, set this to '<Leader>s' instead and nothing is shadowed at all.
let g:surround_lite_prefix = get(g:, 'surround_lite_prefix', 's')

" ----------------------------------------------------------------------------
" The whole table: key -> [open, close, roomy?]
" ----------------------------------------------------------------------------

let s:pairs = {
      \ 'b': ['(', ')', 0], '(': ['(', ')', 1], ')': ['(', ')', 0],
      \ 'B': ['{', '}', 0], '{': ['{', '}', 1], '}': ['{', '}', 0],
      \ 'r': ['[', ']', 0], '[': ['[', ']', 1], ']': ['[', ']', 0],
      \ 'a': ['<', '>', 0], '<': ['<', '>', 1], '>': ['<', '>', 0],
      \ }

let s:quotes = ['"', "'", '`']

" ----------------------------------------------------------------------------
" Input
" ----------------------------------------------------------------------------

function! s:Beep() abort
  execute "normal! \<Esc>"
  return 0
endfunction

" Always show a prompt: a command waiting for input should never look like
" a command that silently did nothing.
function! s:GetChar(prompt) abort
  if a:prompt !=# ''
    echohl ModeMsg | echo a:prompt | echohl None
  endif
  let l:raw = getchar()
  let l:c = (type(l:raw) == type(0)) ? nr2char(l:raw) : l:raw
  redraw | echo ''
  if l:c ==# "\<Esc>" || l:c ==# "\<C-C>"
    return ''
  endif
  return l:c
endfunction

" Reads a target, allowing digits in front of it (ds2( == 2ds().
" Returns [char, count] or ['', 0] when cancelled.
function! s:GetTarget(cnt, prompt) abort
  let l:cnt = a:cnt
  let l:c = s:GetChar(a:prompt)
  let l:digits = ''
  while l:c =~# '^\d$'
    let l:digits .= l:c
    let l:c = s:GetChar(a:prompt . l:digits)
  endwhile
  if l:c ==# ''
    return ['', 0]
  endif
  if l:digits !=# ''
    let l:cnt = l:cnt * str2nr(l:digits)
  endif
  return [l:c, l:cnt]
endfunction

" ----------------------------------------------------------------------------
" What a key means when you WRAP with it: ['open', 'close'] or [] if invalid
" ----------------------------------------------------------------------------

function! s:Delims(ch) abort
  if has_key(s:pairs, a:ch)
    let [l:o, l:c, l:roomy] = s:pairs[a:ch]
    let l:sp = l:roomy ? ' ' : ''
    return [l:o . l:sp, l:sp . l:c]
  elseif a:ch =~# '[[:alnum:]]'
    return []
  elseif a:ch =~# '\p'
    return [a:ch, a:ch]
  endif
  return []
endfunction

" ----------------------------------------------------------------------------
" Reading text out of the buffer by [lnum, col] (inclusive, charwise)
" ----------------------------------------------------------------------------

function! s:TextOf(a, b) abort
  let l:lines = getline(a:a[0], a:b[0])
  if empty(l:lines)
    return ''
  endif
  let l:lines[-1] = strpart(l:lines[-1], 0, a:b[1])
  let l:lines[0]  = strpart(l:lines[0], a:a[1] - 1)
  return join(l:lines, "\n")
endfunction

function! s:CharAt(pos) abort
  return strpart(getline(a:pos[0]), a:pos[1] - 1, 1)
endfunction

" Replace the inclusive charwise region [a..b] with a:text, which may
" contain newlines. Plain line surgery -- no registers, no visual mode.
function! s:ReplaceSpan(a, b, text) abort
  let l:head = strpart(getline(a:a[0]), 0, a:a[1] - 1)
  let l:tail = strpart(getline(a:b[0]), a:b[1])
  let l:new  = split(l:head . a:text . l:tail, "\n", 1)

  call setline(a:a[0], l:new[0])
  if len(l:new) > 1
    call append(a:a[0], l:new[1:])
  endif
  let l:extra = a:b[0] - a:a[0]
  if l:extra > 0
    let l:from = a:a[0] + len(l:new)
    execute 'silent ' . l:from . ',' . (l:from + l:extra - 1) . 'delete _'
  endif
endfunction

" ----------------------------------------------------------------------------
" Finding the surrounding pair
" ----------------------------------------------------------------------------

" Run a text object and report the span it selected, or [] if it did nothing.
function! s:ObjSpan(obj) abort
  let l:pos = getpos('.')
  " Collapse '< and '> onto the cursor first, so a failed object is visible
  " as a zero-width selection rather than leaving stale marks behind.
  execute "silent! normal! v\<Esc>"
  execute 'silent! normal! v' . a:obj
  execute "silent! normal! \<Esc>"
  let l:a = getpos("'<")[1:2]
  let l:b = getpos("'>")[1:2]
  call setpos('.', l:pos)
  return [l:a, l:b]
endfunction

" Returns [[l,c] open, [l,c] close] of the delimiters themselves, or [].
function! s:FindSpan(ch, cnt) abort
  if has_key(s:pairs, a:ch)
    let [l:o, l:c, l:roomy] = s:pairs[a:ch]
    " a( never grabs surrounding whitespace, and it spans lines, so the
    " first and last characters of the span ARE the delimiters.
    let [l:a, l:b] = s:ObjSpan(a:cnt . 'a' . l:o)
    if s:CharAt(l:a) ==# l:o && s:CharAt(l:b) ==# l:c
      return [l:a, l:b]
    endif
    return []
  endif

  if index(s:quotes, a:ch) >= 0
    " a" DOES grab the trailing whitespace, so use i" and step outward.
    let [l:a, l:b] = s:ObjSpan('i' . a:ch)
    let l:open  = [l:a[0], l:a[1] - 1]
    let l:close = [l:b[0], l:b[1] + 1]
    if l:open[1] >= 1 && s:CharAt(l:open) ==# a:ch && s:CharAt(l:close) ==# a:ch
      return [l:open, l:close]
    endif
    " Empty quotes ("") give no inner span; look at the cursor itself.
    return s:ScanLine(a:ch)
  endif

  return s:ScanLine(a:ch)
endfunction

" Arbitrary punctuation has no text object: find the nearest one at or
" before the cursor and its partner after. Single line only.
function! s:ScanLine(ch) abort
  let l:line = getline('.')
  let l:lnum = line('.')
  let l:cur  = col('.') - 1

  " If sitting on the character, it could be either end -- try opening first.
  if strpart(l:line, l:cur, 1) ==# a:ch
    let l:nxt = stridx(l:line, a:ch, l:cur + 1)
    if l:nxt >= 0
      return [[l:lnum, l:cur + 1], [l:lnum, l:nxt + 1]]
    endif
    let l:prv = strridx(l:line, a:ch, l:cur - 1)
    if l:prv >= 0
      return [[l:lnum, l:prv + 1], [l:lnum, l:cur + 1]]
    endif
    return []
  endif

  let l:prv = strridx(l:line, a:ch, l:cur)
  if l:prv < 0
    return []
  endif
  let l:nxt = stridx(l:line, a:ch, l:prv + 1)
  if l:nxt < 0 || l:nxt < l:cur
    return []
  endif
  return [[l:lnum, l:prv + 1], [l:lnum, l:nxt + 1]]
endfunction

" ----------------------------------------------------------------------------
" The three commands
" ----------------------------------------------------------------------------

" Shared: pull out the contents, optionally trimmed, and rebuild.
function! s:Rewrite(ch, cnt, delims) abort
  let l:span = s:FindSpan(a:ch, a:cnt)
  if empty(l:span)
    return s:Beep()
  endif
  let [l:a, l:b] = l:span
  let l:outer = s:TextOf(l:a, l:b)
  let l:inner = strpart(l:outer, 1, strlen(l:outer) - 2)

  " The opening bracket means "roomy", so as a target it takes the spaces
  " back out again. The closing bracket leaves them.
  if has_key(s:pairs, a:ch) && s:pairs[a:ch][2]
    let l:inner = substitute(l:inner, '^\s*\(.\{-}\)\s*$', '\1', '')
  endif

  let l:text = empty(a:delims) ? l:inner : a:delims[0] . l:inner . a:delims[1]
  call s:ReplaceSpan(l:a, l:b, l:text)
  call cursor(l:a[0], l:a[1])
  return 1
endfunction

function! s:Delete() abort
  let [l:ch, l:cnt] = s:GetTarget(v:count1, 'delete surround: ')
  if l:ch ==# ''
    return s:Beep()
  endif
  if s:Rewrite(l:ch, l:cnt, [])
    silent! call repeat#set("\<Plug>(surround-lite-delete)" . l:ch, l:cnt)
  endif
endfunction

function! s:Change() abort
  let [l:ch, l:cnt] = s:GetTarget(v:count1, 'change surround: ')
  if l:ch ==# ''
    return s:Beep()
  endif
  let l:new = s:GetChar('change ' . l:ch . ' into: ')
  if l:new ==# ''
    return s:Beep()
  endif
  let l:d = s:Delims(l:new)
  if empty(l:d)
    return s:Beep()
  endif
  if s:Rewrite(l:ch, l:cnt, l:d)
    silent! call repeat#set(
          \ "\<Plug>(surround-lite-change)" . l:ch . l:new, l:cnt)
  endif
endfunction

" ----------------------------------------------------------------------------
" Adding
" ----------------------------------------------------------------------------

function! s:WrapSpan(a, b, delims) abort
  let l:text = s:TextOf(a:a, a:b)
  call s:ReplaceSpan(a:a, a:b, a:delims[0] . l:text . a:delims[1])
  call cursor(a:a[0], a:a[1])
endfunction

" Whole lines: delimiters go on lines of their own, at the block's indent.
function! s:WrapLines(l1, l2, delims) abort
  let l:indent = matchstr(getline(a:l1), '^\s*')
  let l:open   = substitute(a:delims[0], '\s*$', '', '')
  let l:close  = substitute(a:delims[1], '^\s*', '', '')
  call append(a:l2, l:indent . l:close)
  call append(a:l1 - 1, l:indent . l:open)
  call cursor(a:l1, 1)
endfunction

" ys{motion} -- one function doing both the setup and the operator.
function! s:OpAdd(type, ...) abort
  if a:type ==# 'setup'
    let &operatorfunc = matchstr(expand('<sfile>'), '<SNR>\w\+$')
    return 'g@'
  endif
  let l:ch = s:GetChar('surround with: ')
  let l:d  = s:Delims(l:ch)
  if empty(l:d)
    return s:Beep()
  endif
  if a:type ==# 'line'
    call s:WrapLines(line("'["), line("']"), l:d)
  else
    call s:WrapSpan(getpos("'[")[1:2], getpos("']")[1:2], l:d)
  endif
endfunction

" yss -- the line's own text, ignoring indentation.
function! s:AddLine() abort
  let l:ch = s:GetChar('surround with: ')
  let l:d  = s:Delims(l:ch)
  if empty(l:d)
    return s:Beep()
  endif
  let l:line = getline('.')
  if l:line =~# '^\s*$'
    return s:Beep()
  endif
  let l:c1 = match(l:line, '\S') + 1
  let l:c2 = match(l:line, '\s*$')
  let l:n  = line('.')
  call s:WrapSpan([l:n, l:c1], [l:n, l:c2], l:d)
  silent! call repeat#set("\<Plug>(surround-lite-line)" . l:ch)
endfunction

function! s:AddVisual() abort
  let l:mode = visualmode()
  let l:ch = s:GetChar('surround selection with: ')
  let l:d  = s:Delims(l:ch)
  if empty(l:d)
    return s:Beep()
  endif
  if l:mode ==# 'V'
    call s:WrapLines(line("'<"), line("'>"), l:d)
  elseif l:mode ==# 'v'
    call s:WrapSpan(getpos("'<")[1:2], getpos("'>")[1:2], l:d)
  else
    echohl WarningMsg
    echomsg 'surround-lite: blockwise selections are not supported'
    echohl None
    return s:Beep()
  endif
endfunction

" ----------------------------------------------------------------------------
" Mappings
" ----------------------------------------------------------------------------

nnoremap <silent> <Plug>(surround-lite-delete) :<C-u>call <SID>Delete()<CR>
nnoremap <silent> <Plug>(surround-lite-change) :<C-u>call <SID>Change()<CR>
nnoremap <silent> <Plug>(surround-lite-line)   :<C-u>call <SID>AddLine()<CR>
nnoremap <expr>   <Plug>(surround-lite-add)    <SID>OpAdd('setup')
xnoremap <silent> <Plug>(surround-lite-visual) :<C-u>call <SID>AddVisual()<CR>

if !g:surround_lite_no_mappings
  let s:p = g:surround_lite_prefix
  execute 'nmap ' . s:p . 'a <Plug>(surround-lite-add)'
  execute 'nmap ' . s:p . 'd <Plug>(surround-lite-delete)'
  execute 'nmap ' . s:p . 'c <Plug>(surround-lite-change)'
  execute 'nmap ' . s:p . 's <Plug>(surround-lite-line)'
  execute 'xmap ' . s:p . '  <Plug>(surround-lite-visual)'
  unlet s:p
endif

let &cpo = s:save_cpo
unlet s:save_cpo

" ============================================================================
" NOTES
"
" Dropped on purpose (vim-surround has them, this does not):
"   t T <C-T> tags, l \ LaTeX, f F <C-F> functions, <C-]> brace blocks,
"   p paragraphs, the insert-mode <C-S> / <C-G>s mappings, cS yS ySs gS,
"   g:surround_{N} custom definitions, automatic re-indenting.
"
" Bindings
"   sa{motion}{d}   add       sd{d}  delete       sc{old}{new}  change
"   ss{d}           whole line              {visual} s{d}  the selection
"
"   The prefix is g:surround_lite_prefix, default "s". It shadows normal-mode
"   s (substitute one character); cl is the exact equivalent. If you would
"   rather keep s, put this in your config BEFORE sourcing this file:
"       let g:surround_lite_prefix = '<Leader>s'
"   and the commands become <Leader>sa, <Leader>sd, <Leader>sc, <Leader>ss
"   and <Leader>s in visual mode, shadowing nothing at all.
"
"   In visual mode the prefix alone is the command, so "s" there shadows
"   v_s (change the selection) -- c does the identical thing.
"
" Select mode is NOT mapped. If you have 'selectmode' or 'keymodel' set so
" that Shift-Arrow starts a selection, that selection is a SELECT-mode one,
" where typing a letter REPLACES the selection. Select with v or V instead,
" or check:  :set selectmode? keymodel?
"
" Dot-repeat for sd, sc and ss works when tpope/vim-repeat is installed.
" Without it "." re-prompts for the character, which is still usable.
"
" Blockwise visual selections are refused rather than mangled.
" ============================================================================

