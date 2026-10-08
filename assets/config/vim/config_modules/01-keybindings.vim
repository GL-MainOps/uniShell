" vim: set ft=vim:
" ╔═══════════════════════════════════════════════════════════════
" ║ KEYBINDINGS

" ╭────────────────────────────────╮
" │ MISC                           │
" ╰────────────────────────────────╯
" @cheat-group MISC
" @cheat jj                  <Esc>                      | Exit insert mode quickly
" @cheat <F2>                                           | Toggle paste Mode
" @cheat <leader>t           :ter<CR>                   | OPEN TERMINAL IN SPLIT
" @cheat <leader>h           :set hlsearch!<CR>         | TOGGLE SEARCH HIGHLIGHTING

set pastetoggle=<F2>
inoremap jj <Esc>
nnoremap <silent> <leader>t :ter<CR>
nnoremap <silent> <leader>h :set hlsearch!<CR>
xnoremap p P



" ╭────────────────────────────────╮
" │ PLUGINS                        │
" ╰────────────────────────────────╯
" @cheat-group FZF
" @cheat <leader>fb          :Buffers<CR>               | BROWSE OPENED BUFFERS
" @cheat <leader>ff          :Files<CR>                 | BROWSE FILES IN CWD
" @cheat <leader>fg          :GFiles<CR>                | BROWSE GIT FILES
" @cheat <leader>fs/fl       :BLines<CR>                | BROWSE LINES OF CURRENT BUFFER
" @cheat <leader>fr          :Rg<CR>                    | GREP A [pattern] in CWD
" @cheat <leader>fk          :Maps<CR>                  | BROWSE KEYBINDINGS

nnoremap <leader>fb :Buffers<CR>
nnoremap <leader>ff :Files<CR>
nnoremap <leader>fg :GFiles<CR>
nnoremap <leader>fs :BLines<CR>
nnoremap <leader>fl :BLines<CR>
nnoremap <leader>fr :Rg<CR>
nnoremap <leader>fk :Maps<CR>



" ╭────────────────────────────────╮
" │ FILE OPERATIONS                │
" ╰────────────────────────────────╯
" @cheat-group FILE OPERATIONS
" @cheat <leader>w           :w<CR>                     | SAVE CURRENT FILE
" @cheat <leader>e           :E<CR>                     | OPEN FILE EXPLORER (NETRW)
" @cheat QUITTING VIM
" @cheat <silent>            <leader>q  :q<CR>          | QUIT VIM (NOT ALL BUFFERS)
" @cheat <silent>            <leader>Q  :qa<CR>         | QUIT VIM (ALL BUFFERS)

nnoremap <silent> <leader>w :w<CR>
nnoremap <silent> <leader>e :E<CR>
nnoremap <silent> <leader>q :q<CR>
nnoremap <silent> <leader>Q :qa<CR>



" ╭────────────────────────────────╮
" │ BUFFER MANAGEMENT              │
" ╰────────────────────────────────╯
" @cheat-group BUFFER MANAGEMENT
" @cheat <leader><space>     :b#<CR>                    | CREATE NEW EMPTY BUFFER
" @cheat <leader><Backspace> :enew<CR>                  | SWITCH TO LAST USED BUFFER

noremap <silent> <leader><space> :b#<CR>
noremap <silent> <leader><Backspace> :enew<CR>



" ╭────────────────────────────────╮
" │ WINDOW MANAGEMENT              │
" ╰────────────────────────────────╯
" @cheat-group WINDOW MANAGEMENT
" @cheat MOVE BETWEEN WINDOWS with <leader>+ARROW KEY
" @cheat <leader><Left>      <C-w>h                     |
" @cheat <leader><Down>      <C-w>j                     |
" @cheat <leader><Up>        <C-w>k                     |
" @cheat <leader><Right>     <C-w>l                     |
" @cheat Resize with <leader>+SHIFT+ARROWS, in steps of 3 (a count multiplies it)
" @cheat <leader><S-Up>      <leader>+                  |
" @cheat <leader><S-Down>    <leader>-                  |
" @cheat <leader><S-Right>   <leader>>                  |
" @cheat <leader><S-Left>    <leader><                  |
" @cheat SPLIT MANAGEMENT
" @cheat <leader>=           <C-w>=                     | MAKE ALL WINDOWS EQUAL SIZE
" @cheat <silent>            <leader>ss :split<CR>      | CREATE HORIZONTAL SPLIT (──)
" @cheat <silent>            <leader>sv :vsplit<CR>     | CREATE VERTICAL SPLIT (│)
" @cheat <silent>            <leader>sx :close<CR>      | CLOSE CURRENT WINDOW

nnoremap <leader><Left>  <C-w>h
nnoremap <leader><Down>  <C-w>j
nnoremap <leader><Up>    <C-w>k
nnoremap <leader><Right> <C-w>l

nnoremap <silent> <leader>+ :<C-u>execute 'resize +'          . 3 * v:count1<CR>
nnoremap <silent> <leader>- :<C-u>execute 'resize -'          . 3 * v:count1<CR>
nnoremap <silent> <leader>> :<C-u>execute 'vertical resize +' . 3 * v:count1<CR>
nnoremap <silent> <leader>< :<C-u>execute 'vertical resize -' . 3 * v:count1<CR>
nmap <leader><S-Up>    <leader>+
nmap <leader><S-Down>  <leader>-
nmap <leader><S-Right> <leader>>
nmap <leader><S-Left>  <leader><

nnoremap <leader>= <C-w>=
nnoremap <silent> <leader>ss :split<CR>
nnoremap <silent> <leader>sv :vsplit<CR>
nnoremap <silent> <leader>sx :close<CR>



" ╭────────────────────────────────╮
" │ EDITING                        │
" ╰────────────────────────────────╯
" @cheat-group MOTION
" @cheat <C-a>  <Home> | JUMP TO START OF LINE (ALL MODES)
" @cheat <C-e>  <End>  | JUMP TO END OF LINE  ; KEEPS THE <C-v>$ ragged-block flag

" Home/End everywhere, including insert and command line
noremap  <C-a> <Home>
noremap  <C-e> <End>
inoremap <C-a> <Home>
inoremap <C-e> <End>
cnoremap <C-a> <Home>


" @cheat-group EDITING
" @cheat <Leader>+ <Leader>- | increment / decrement number (was <C-a>/<C-x>)
" @cheat p  (visual)                                    | replace selection, keep your yank (native v_P)
" @cheat <Leader>p                                      | old swap behaviour: selection goes into the register

" rescue increment / decrement
nnoremap <Leader>+ <C-a>
nnoremap <Leader>- <C-x>
xnoremap <Leader>+ <C-a>
xnoremap <Leader>- <C-x>
" ┌────────────────┐
" │ FORMATTING     │ BORROWED FROM defaults.vim
" └────────────────┘
" Revert with ":unmap Q".
map Q gq
sunmap Q

" CTRL-U in insert mode deletes a lot.  Use CTRL-G u to first break undo, abcd
" efgh ijklm qwerty q
" so that you can undo CTRL-U after inserting a line break.
" Revert with ":iunmap <C-U>".
inoremap <C-U> <C-G>u<C-U>

" ╚═══════════════════════════════════════════════════════════════
