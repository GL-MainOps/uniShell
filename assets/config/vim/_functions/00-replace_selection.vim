" vim: set ft=vim:
" ╭────────────────────────────────╮
" │ REPLACE SELECTION / WORD       │
" ╰────────────────────────────────╯
"   v_<leader>r   replace every occurrence of the selected text
"   v_<leader>R   same, but whole words only
"   n_<leader>r   replace every whole-word occurrence of the word under cursor
"
" Each opens ":%s/<pattern>//gc" with the cursor in the replacement field.
" The text is matched literally (\V) and case-sensitively (\C), so . * [ ] ~ ^ $
" \ / and multi-line selections all just work. Registers, cursor position and
" scroll are left untouched. <Esc> on the command line cancels.
" Tip: type a range in place of % with <C-b><Del> before confirming, or set
" g:replace_flags = 'g' to skip the per-match confirmation.

let g:replace_flags = get(g:, 'replace_flags', 'gc')

" Build the :s command (no leading colon) for a piece of literal text
function! s:Build(text, whole) abort
  " \V makes everything literal except \ and /, and ^ $ at the pattern's edges;
  " \%x5e / \%x24 are a literal ^ / $ in any position
  let l:pat = escape(a:text, '\/')
  let l:pat = substitute(l:pat, '\^', '\\%x5e', 'g')
  let l:pat = substitute(l:pat, '\$', '\\%x24', 'g')
  let l:pat = substitute(l:pat, "\n", '\\n', 'g')
  let l:pat = substitute(l:pat, "\r", '\\r', 'g')
  let l:pat = substitute(l:pat, "\t", '\\t', 'g')
  " other control chars (e.g. a literal <Esc>) must not act as keys when typed
  let l:pat = substitute(l:pat, '[[:cntrl:]]', "\<C-v>&", 'g')
  if a:whole
    " word boundaries only where the text actually starts/ends with a word char
    if a:text =~# '^\k' | let l:pat = '\<' . l:pat | endif
    if a:text =~# '\k$' | let l:pat = l:pat . '\>' | endif
  endif
  return '%s/\V\C' . l:pat . '//' . g:replace_flags
endfunction

" Put the command on the command line with the cursor between the last slashes
function! s:Replace(text, whole) abort
  if a:text ==# ''
    return
  endif
  call feedkeys(':' . s:Build(a:text, a:whole)
        \ . repeat("\<Left>", strchars(g:replace_flags) + 1), 'n')
endfunction

function! s:ReplaceVisual(whole) abort
  let l:view = winsaveview()
  let l:save = [getreg('z'), getregtype('z')]
  silent normal! gv"zy
  let [l:text, l:type] = [getreg('z'), getregtype('z')]
  call setreg('z', l:save[0], l:save[1])
  call winrestview(l:view)
  if l:type ==# 'V'                           " linewise: drop the trailing newline
    let l:text = substitute(l:text, '\n$', '', '')
  endif
  call s:Replace(l:text, a:whole)
endfunction

" @cheat-group FUNCTIONS
" @cheat <leader>r (VISUAL)              | REPLACE EVERY OCCURRENCE OF THE SELECTED TEXT
" @cheat <leader>r (NORMAL)              | REPLACE EVERY WHOLE-WORD OCCURRENCE OF THE WORD UNDER CURSOR
" @cheat <leader>R                       | SAME, BUT WHOLE WORDS ONLY

" xnoremap (not vnoremap) so these never fire in Select mode
xnoremap <silent> <leader>r :<C-u>call <SID>ReplaceVisual(0)<CR>
xnoremap <silent> <leader>R :<C-u>call <SID>ReplaceVisual(1)<CR>
nnoremap <silent> <leader>r :<C-u>call <SID>Replace(expand('<cword>'), 1)<CR>
