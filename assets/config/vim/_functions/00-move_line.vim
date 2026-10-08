" vim: set ft=vim:
" @cheat-group MOTION
" @cheat MOVE ── NORMAL = CURSOR LINE, VISUAL = SELECTION (KEPT)
" @cheat <M-Up> OR <M-k>                                | MOVE LINE UP (VISUAL & NORMAL)
" @cheat <<M-Down> OR <M-j>                             | MOVE LINE DOWN (VISUAL & NORMAL)
" @cheat DUPLICATE ── COUNT = NUMBER OF COPIES; CURSOR LANDS ON THE COPY
" @cheat <M-S-Up> OR <Leader>D                          | DUPLICATE UP
" @cheat <M-S-Down> OR <Leader>d                        | DUPLICATE DOWN
" @cheat INDENT ── SELECTION IS KEPT, SO YOU CAN REPEAT
" @cheat <TAB> AND <S-TAB>                              | ADD OR REMOVE INDENTATION
" @cheat BLANK LINES ── CURSOR DOES NOT MOVE, STAYS IN NORMAL MODE
" @cheat <Leader>O                                      | ADD BLANK LINES ABOVE
" @cheat <Leader>o                                      | ADD BLANK LINES BELOW



" ============================================================================
" move-lines.vim -- move and duplicate lines like a modern editor
"
" Install:  ~/.vim/plugin/move-lines.vim
"     or:   source ~/path/to/move-lines.vim   " from your vimrc
"
" Works in Vim 8.x, Vim 9.x and Neovim. No dependencies.
" ----------------------------------------------------------------------------
" CHEATSHEET                      all of these take a count, e.g. 3<M-j>
"
"   MOVE              normal = cursor line, visual = selection (kept)
"     <M-Up>   <M-k>   [e        move up
"     <M-Down> <M-j>   ]e        move down
"     <M-Up>/<M-Down> also work in insert mode
"
"   DUPLICATE         count = number of copies; cursor lands on the copy
"     <M-S-Up>         <Leader>D  duplicate above
"     <M-S-Down>       <Leader>d  duplicate below
"
"   INDENT            selection is kept, so you can repeat
"     <  >  <Tab>  <S-Tab>
"
"   BLANK LINES       cursor does not move, stays in normal mode
"     <Leader>o                   add line(s) below
"     <Leader>O                   add line(s) above
"
"   DELETE
"     <M-S-k>                     delete line to black hole (keeps your yank)
"
"   [e / ]e / <Leader>d / <Leader>D are the terminal-safe fallbacks --
"   use them if Alt keys aren't coming through (see NOTES at the bottom).
" ============================================================================
 
if exists('g:loaded_move_lines') || &compatible
  finish
endif
let g:loaded_move_lines = 1
 
let s:save_cpo = &cpo
set cpo&vim
 
" ----------------------------------------------------------------------------
" Options -- set these in your vimrc BEFORE this file is sourced
" ----------------------------------------------------------------------------
 
" Re-indent lines after moving them.
"   0 = never touch indentation (default; matches VS Code, non-destructive)
"   1 = run '=' over the moved lines (the old `gv=gv` behaviour)
let g:move_lines_reindent = get(g:, 'move_lines_reindent', 0)
 
" Install the default key mappings. Set to 0 to map things yourself.
let g:move_lines_default_maps = get(g:, 'move_lines_default_maps', 1)
 
" Teach terminal Vim the Alt-key escape sequences (see note at bottom).
" Not needed in gvim / MacVim / Neovim. Off by default because it makes Vim
" wait &ttimeoutlen ms after a bare <Esc>.
let g:move_lines_fix_alt_keys = get(g:, 'move_lines_fix_alt_keys', 0)
 
" ----------------------------------------------------------------------------
" Internals
" ----------------------------------------------------------------------------
 
function! s:Writable() abort
  if !&modifiable || &readonly
    echohl ErrorMsg
    echomsg 'move-lines: buffer is not modifiable'
    echohl None
    return 0
  endif
  return 1
endfunction
 
" Re-indent [first,last]. No-op unless g:move_lines_reindent is set.
function! s:Reindent(first, last) abort
  if !g:move_lines_reindent
    return
  endif
  let l:view = winsaveview()
  execute 'silent! keepjumps normal! ' . a:first . 'G=' . a:last . 'G'
  call winrestview(l:view)
endfunction
 
" Move lines [first,last] by {cnt} in direction {dir} (-1 up, +1 down).
" Clamps at both ends of the buffer. Returns [new_first, new_last], or []
" when nothing could move.
function! s:MoveBlock(first, last, dir, cnt) abort
  if !s:Writable()
    return []
  endif
  let l:eof    = line('$')
  let l:height = a:last - a:first + 1
 
  if a:dir < 0
    if a:first <= 1
      return []
    endif
    let l:dest      = max([0, a:first - a:cnt - 1])
    let l:new_first = l:dest + 1
  else
    if a:last >= l:eof
      return []
    endif
    let l:dest      = min([l:eof, a:last + a:cnt])
    let l:new_first = l:dest - l:height + 1
  endif
 
  execute a:first . ',' . a:last . 'move ' . l:dest
  return [l:new_first, l:new_first + l:height - 1]
endfunction
 
" Copy lines [first,last] {cnt} times, above (dir < 0) or below (dir > 0).
" Returns [new_first, new_last] spanning the copies, or [].
function! s:CopyBlock(first, last, dir, cnt) abort
  if !s:Writable()
    return []
  endif
  let l:height = a:last - a:first + 1
  let l:dest   = a:dir < 0 ? a:first - 1 : a:last
 
  " Same source range every time: when copying up the copy lands on
  " [first,last] and is byte-identical, so repeating is safe.
  for l:i in range(a:cnt)
    execute a:first . ',' . a:last . 'copy ' . l:dest
  endfor
 
  let l:new_first = a:dir < 0 ? a:first : a:last + 1
  return [l:new_first, l:new_first + (l:height * a:cnt) - 1]
endfunction
 
" ----------------------------------------------------------------------------
" Normal mode -- operates on the cursor line. Accepts a count.
" ----------------------------------------------------------------------------
 
function! s:MoveNormal(dir) abort
  let l:cnt = v:count1
  let l:col = col('.')
  let l:ln  = line('.')
  let l:new = s:MoveBlock(l:ln, l:ln, a:dir, l:cnt)
  if empty(l:new)
    return
  endif
  call s:Reindent(l:new[0], l:new[1])
  call cursor(l:new[0], l:col)
  silent! normal! zv
endfunction
 
" count = number of copies to make.
function! s:DupNormal(dir) abort
  let l:cnt = v:count1
  let l:col = col('.')
  let l:ln  = line('.')
  let l:new = s:CopyBlock(l:ln, l:ln, a:dir, l:cnt)
  if empty(l:new)
    return
  endif
  " Land on the new copy so you can start editing it immediately.
  call cursor(l:new[0], l:col)
  silent! normal! zv
endfunction
 
" ----------------------------------------------------------------------------
" Visual mode -- operates on the selected lines, keeps the selection.
" ----------------------------------------------------------------------------
 
function! s:MoveVisual(dir) abort
  let l:cnt = v:count1
  let l:new = s:MoveBlock(line("'<"), line("'>"), a:dir, l:cnt)
  if empty(l:new)
    normal! gv
    return
  endif
  call s:Reindent(l:new[0], l:new[1])
  " ':move' carries the '< and '> marks along with the lines, so gv
  " reselects the block in its original mode (charwise/linewise/blockwise).
  normal! gv
endfunction
 
function! s:DupVisual(dir) abort
  let l:cnt = v:count1
  let l:new = s:CopyBlock(line("'<"), line("'>"), a:dir, l:cnt)
  if empty(l:new)
    normal! gv
    return
  endif
  " Select the copies explicitly -- linewise, since that's what we made.
  execute 'normal! ' . l:new[0] . 'GV' . l:new[1] . 'G'
endfunction
 
" ----------------------------------------------------------------------------
" Plug mappings -- remap these if you don't want the defaults
" ----------------------------------------------------------------------------
 
nnoremap <silent> <Plug>(move-lines-up)    :<C-u>call <SID>MoveNormal(-1)<CR>
nnoremap <silent> <Plug>(move-lines-down)  :<C-u>call <SID>MoveNormal(1)<CR>
nnoremap <silent> <Plug>(dup-lines-up)     :<C-u>call <SID>DupNormal(-1)<CR>
nnoremap <silent> <Plug>(dup-lines-down)   :<C-u>call <SID>DupNormal(1)<CR>
 
xnoremap <silent> <Plug>(move-lines-up)    :<C-u>call <SID>MoveVisual(-1)<CR>
xnoremap <silent> <Plug>(move-lines-down)  :<C-u>call <SID>MoveVisual(1)<CR>
xnoremap <silent> <Plug>(dup-lines-up)     :<C-u>call <SID>DupVisual(-1)<CR>
xnoremap <silent> <Plug>(dup-lines-down)   :<C-u>call <SID>DupVisual(1)<CR>
 
inoremap <silent> <Plug>(move-lines-up)    <C-o>:call <SID>MoveNormal(-1)<CR>
inoremap <silent> <Plug>(move-lines-down)  <C-o>:call <SID>MoveNormal(1)<CR>
 
" ----------------------------------------------------------------------------
" Default key mappings
" ----------------------------------------------------------------------------
 
if g:move_lines_default_maps
 
  if g:move_lines_fix_alt_keys && !has('gui_running') && !has('nvim')
    set ttimeout
    if &ttimeoutlen <= 0
      set ttimeoutlen=50
    endif
    for s:key in ['j', 'k']
      execute "set <M-" . s:key . ">=\<Esc>" . s:key
    endfor
    unlet! s:key
  endif
 
  " --- Move: Alt-Up/Down and Alt-k/j (VS Code parity) ---
  for s:mode in ['n', 'x', 'i']
    execute s:mode . 'map <M-Up>   <Plug>(move-lines-up)'
    execute s:mode . 'map <M-Down> <Plug>(move-lines-down)'
  endfor
  unlet! s:mode
 
  nmap <M-k> <Plug>(move-lines-up)
  nmap <M-j> <Plug>(move-lines-down)
  xmap <M-k> <Plug>(move-lines-up)
  xmap <M-j> <Plug>(move-lines-down)
 
  " --- Duplicate: Shift-Alt-Up/Down (VS Code parity) ---
  nmap <M-S-Up>   <Plug>(dup-lines-up)
  nmap <M-S-Down> <Plug>(dup-lines-down)
  xmap <M-S-Up>   <Plug>(dup-lines-up)
  xmap <M-S-Down> <Plug>(dup-lines-down)
 
  " --- Terminal-safe fallbacks: these always work, no Alt required ---
  nmap [e <Plug>(move-lines-up)
  nmap ]e <Plug>(move-lines-down)
  xmap [e <Plug>(move-lines-up)
  xmap ]e <Plug>(move-lines-down)
 
  nmap <Leader>D <Plug>(dup-lines-up)
  nmap <Leader>d <Plug>(dup-lines-down)
  xmap <Leader>D <Plug>(dup-lines-up)
  xmap <Leader>d <Plug>(dup-lines-down)
 
endif
 
" ============================================================================
" Companion mappings -- small quality-of-life wins in the same spirit.
" Delete any you don't want.
" ============================================================================
 
" Keep the selection after indenting, so you can press < / > repeatedly.
xnoremap < <gv
xnoremap > >gv
 
" Tab / Shift-Tab to indent a selection (shadows <Tab> as a visual motion).
xnoremap <Tab>   >gv
xnoremap <S-Tab> <gv
 
" Insert a blank line above/below without leaving normal mode or moving.
" Takes a count: 3<Leader>o adds three blank lines below.
nnoremap <silent> <Leader>o :<C-u>call append(line('.'),   repeat([''], v:count1))<CR>
nnoremap <silent> <Leader>O :<C-u>call append(line('.')-1, repeat([''], v:count1))<CR>

" Delete the current line without clobbering the unnamed register
" (VS Code's Ctrl-Shift-K).
nnoremap <silent> <M-S-k> "_dd
xnoremap <silent> <M-S-k> "_d

" Make <Esc>-prefixed sequences resolve quickly without delaying <Esc> itself.
set timeout
set ttimeout
if &ttimeoutlen <= 0
  set ttimeoutlen=50
endif

let &cpo = s:save_cpo
unlet s:save_cpo

" ============================================================================
" NOTES
"
" Alt keys in terminal Vim
"   gvim, MacVim and Neovim handle <M-...> natively -- nothing to do.
"   Terminal Vim often doesn't. Check with:  :verbose map <M-k>
"   and by typing  <C-v><M-k>  in insert mode (you should see ^[k, not just k).
"   If it's not working, either:
"     a) configure your terminal to send Esc+key for Alt
"        (iTerm2: "Esc+" for Left Option; Alacritty/kitty do this already),
"        then set  let g:move_lines_fix_alt_keys = 1  in your vimrc; or
"     b) just use the [e / ]e and <Leader>d fallbacks, which always work.
"   <M-S-Up> is rarely recognised by terminal Vim at all -- use <Leader>d/D.
"
" Why not <S-Up>/<S-Down>?
"   They work fine as terminal keys, but Shift-Arrow is Vim's default
"   scroll-by-page in normal/visual mode and starts a selection in insert mode
"   when 'keymodel' contains "startsel". Remapping them is fine if you don't
"   use either -- add:
"     nmap <S-Up> <Plug>(move-lines-up)      " etc.
"
" Counts
"   3<M-j>  moves the line/selection down 3 lines (clamped at the buffer end).
"   3<Leader>d  makes 3 copies.
"
" Undo
"   Each press is its own undo step: five moves need five u's.
"
" See also: :help :move, :help :copy, :help gv, :help 'ttimeoutlen'
" ============================================================================

