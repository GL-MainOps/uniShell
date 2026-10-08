" vim: set ft=vim:
" ============================================================================
" case.vim -- intuitive upper/lower case control
"
" Install:  <your vimconfig>/plugin/case.vim
"
" Mnemonic: the case of the key is the case you get.
"     <Leader>u  lowercase        <Leader>U  UPPERCASE
"     <Leader>c  Capitalize       <Leader>~  cycle through all three
"
" Every mapping works the same way in both modes:
"     normal mode -> the word under the cursor
"     visual mode -> the selection (charwise, linewise or blockwise)
" ----------------------------------------------------------------------------
" @cheat-group EDITING
" @cheat CASE: Native, for a motion: gu{motion} gU{motion} g~{motion}, guu gUU g~~
" @cheat <Leader>u | lowercase word / selection
" @cheat <Leader>U | UPPERCASE word / selection
" @cheat <Leader>c | Capitalize word, Title Case A Selection
" @cheat <Leader>~ | cycle lower -> Title -> UPPER -> lower
" @cheat In visual mode plain u / U / ~ also work (no leader needed).
" ============================================================================

if exists('g:loaded_case_maps') || &compatible
  finish
endif
let g:loaded_case_maps = 1

let s:save_cpo = &cpo
set cpo&vim

let g:case_default_maps = get(g:, 'case_default_maps', 1)

" ----------------------------------------------------------------------------
" Transforms (pure string in, string out)
" ----------------------------------------------------------------------------

function! s:Lower(s) abort
  return tolower(a:s)
endfunction

function! s:Upper(s) abort
  return toupper(a:s)
endfunction

" First character up, the rest down -- matchstr/strpart rather than [0] so
" this is correct for multibyte leading characters too.
function! s:CapWord(w) abort
  let l:first = matchstr(a:w, '^.')
  return toupper(l:first) . tolower(strpart(a:w, strlen(l:first)))
endfunction

" A word is a run of 'iskeyword' characters, plus any apostrophe sitting
" between two of them. That way don't -> Don't (not Don'T), while a quoted
" 'hello' still becomes 'Hello' rather than being left alone.
function! s:Title(s) abort
  return substitute(a:s, "\\k\\+\\%('\\k\\+\\)*",
        \ '\=s:CapWord(submatch(0))', 'g')
endfunction

" lower -> Title -> UPPER -> lower.  Anything mixed drops to lower.
function! s:Cycle(s) abort
  if a:s ==# tolower(a:s)
    return s:Title(a:s)
  elseif a:s ==# s:Title(a:s)
    return toupper(a:s)
  elseif a:s ==# toupper(a:s)
    return tolower(a:s)
  endif
  return tolower(a:s)
endfunction

" ----------------------------------------------------------------------------
" Applying a transform to the last visual selection
" ----------------------------------------------------------------------------

" Yank the selection, transform it, put it back. Going through a register
" keeps charwise / linewise / blockwise all working without special cases.
function! s:Apply(fn) abort
  let l:save_z    = getreg('z')
  let l:save_ztyp = getregtype('z')
  " A yank into a named register ALSO overwrites the unnamed register, so
  " that one has to be saved and put back too.
  let l:save_q    = getreg('"')
  let l:save_qtyp = getregtype('"')
  let l:save_sel  = &selection
  set selection=inclusive

  normal! gv"zy
  call setreg('z', call(a:fn, [getreg('z')]), getregtype('z'))
  " v_P replaces the selection without disturbing any register.
  normal! gv"zP

  let &selection = l:save_sel
  call setreg('z', l:save_z, l:save_ztyp)
  call setreg('"', l:save_q, l:save_qtyp)
endfunction

function! s:Visual(fn) abort
  call s:Apply(a:fn)
endfunction

" Normal mode: select the word under the cursor, transform, restore the view.
function! s:Word(fn) abort
  let l:view = winsaveview()
  execute "normal! viw\<Esc>"
  call s:Apply(a:fn)
  call winrestview(l:view)
endfunction

" ----------------------------------------------------------------------------
" Plug mappings
" ----------------------------------------------------------------------------

nnoremap <silent> <Plug>(case-lower) :<C-u>call <SID>Word('<SID>Lower')<CR>
nnoremap <silent> <Plug>(case-upper) :<C-u>call <SID>Word('<SID>Upper')<CR>
nnoremap <silent> <Plug>(case-title) :<C-u>call <SID>Word('<SID>Title')<CR>
nnoremap <silent> <Plug>(case-cycle) :<C-u>call <SID>Word('<SID>Cycle')<CR>

xnoremap <silent> <Plug>(case-lower) :<C-u>call <SID>Visual('<SID>Lower')<CR>
xnoremap <silent> <Plug>(case-upper) :<C-u>call <SID>Visual('<SID>Upper')<CR>
xnoremap <silent> <Plug>(case-title) :<C-u>call <SID>Visual('<SID>Title')<CR>
xnoremap <silent> <Plug>(case-cycle) :<C-u>call <SID>Visual('<SID>Cycle')<CR>

" ----------------------------------------------------------------------------
" Default mappings
" ----------------------------------------------------------------------------

if g:case_default_maps
  nmap <Leader>u <Plug>(case-lower)
  nmap <Leader>U <Plug>(case-upper)
  nmap <Leader>c <Plug>(case-title)
  nmap <Leader>~ <Plug>(case-cycle)

  xmap <Leader>u <Plug>(case-lower)
  xmap <Leader>U <Plug>(case-upper)
  xmap <Leader>c <Plug>(case-title)
  xmap <Leader>~ <Plug>(case-cycle)

  " Word-processor muscle memory, if your terminal passes it through.
  " nmap <S-F3> <Plug>(case-cycle)
  " xmap <S-F3> <Plug>(case-cycle)
endif

let &cpo = s:save_cpo
unlet s:save_cpo

" ============================================================================
" NOTES
"
" Dot-repeat
"   These go through :call, so "." does not repeat them. For a repeatable
"   one-off use the native operators instead: gUiw, guiw, g~iw -- all three
"   are dot-repeatable out of the box.
"
" What counts as a word
"   Title Case uses 'iskeyword', which includes "_" and digits by default,
"   so foo_bar becomes Foo_bar rather than Foo_Bar. That is usually right
"   for prose and leaves snake_case identifiers recognisable.
"
" Cycling mixed case
"   camelCase is "mixed", so one cycle flattens it to lowercase. The cycle is
"   for prose, not for renaming identifiers.
" ============================================================================
