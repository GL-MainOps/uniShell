" vim: set ft=vim:
" ╔═══════════════════════════════════════════════════════════════
" ║ INITIAL VIM ENVIRONMENT PREPARATIONS

if &compatible
    set nocompatible
endif

" ╭────────────────────────────────╮
" │ SETTING LEADER KEY EARLY       │
" ╰────────────────────────────────╯
" Set leader key to space
let mapleader="\<space>"
nnoremap <Space> <Nop>


" ╭────────────────────────────────╮
" │ ENABLE CHEATSHEET TO SCAN DIR  │
" ╰────────────────────────────────╯
let g:cheat_paths = [expand('<sfile>:p:h')]


" ╭────────────────────────────────╮
" │ CHANGING SURROUND PREFIX       │
" ╰────────────────────────────────╯
let g:surround_lite_prefix = '<Leader>s'




" ╭────────────────────────────────╮
" │ CACHE DIRECTORIES              │
" ╰────────────────────────────────╯
let s:fallback_root = '/var/tmp/.vimcache'
let s:prune_days    = 90                    " 0 disables pruning

function! s:Warn(msg) abort
    echohl WarningMsg | echomsg 'vimrc: ' . a:msg | echohl None
endfunction

" Name of the user actually running Vim (not whatever $USER claims after
" `sudo -E` / `su`).
function! s:UserName() abort
    let l:name = ''
    if executable('id')
        let l:name = substitute(system('id -un 2>/dev/null'), '\n\+$', '', '')
        if v:shell_error
            let l:name = ''
        endif
    endif
    if empty(l:name)
        let l:name = !empty($USER) ? $USER : $LOGNAME
    endif
    " Make it safe as a single path component.
    let l:name = substitute(l:name, '[^A-Za-z0-9._-]', '_', 'g')
    let l:name = substitute(l:name, '^\.\+', '_', '')
    return empty(l:name) ? 'unknown' : l:name
endfunction

" Trim, require an absolute path, make it canonical without a trailing slash.
" Returns '' for empty, relative, or "/" input.
function! s:NormalizeRoot(path) abort
    let l:path = substitute(a:path, '^\s\+\|\s\+$', '', 'g')
    if l:path !~# '^/'
        return ''
    endif
    let l:path = substitute(fnamemodify(l:path, ':p'), '/\+$', '', '')
    return l:path
endfunction

" Create the shared root if missing: world read/write/exec plus the sticky
" bit, so every user can create their own subdir but can't remove or rename
" anyone else's. Permissions are only touched when this call created it.
" Returns 1 if the directory exists afterwards. Writability is not required
" here: a user's own subdir may already exist and be usable.
function! s:EnsureSharedRoot(root) abort
    if isdirectory(a:root)
        return 1
    endif
    try
        call mkdir(a:root, 'p', 0777)
    catch
        " may have lost a race with another user; re-checked below
    endtry
    if !isdirectory(a:root)
        return 0
    endif
    " mkdir()'s mode is filtered by umask, so set it explicitly.
    call setfperm(a:root, 'rwxrwxrwx')
    if executable('chmod')
        " setfperm() can't set the sticky bit
        call system('chmod 1777 ' . shellescape(a:root) . ' 2>/dev/null')
    endif
    return 1
endfunction

" Create <path>/{swap,undo,backup} (owned by the current user, mode 0700)
" and verify they're writable. Returns <path>, or '' on failure.
function! s:PrepareDir(path) abort
    try
        for l:sub in ['', '/swap', '/undo', '/backup']
            let l:d = a:path . l:sub
            if !isdirectory(l:d)
                call mkdir(l:d, 'p', 0700)
            endif
            if filewritable(l:d) != 2       " 2 == writable directory
                return ''
            endif
        endfor
    catch
        return ''
    endtry
    return a:path
endfunction

" <root>/<user>/{swap,undo,backup}, creating the shared root if needed.
" Returns the per-user directory, or '' on failure.
function! s:PrepareUserDir(root, user) abort
    let l:root = s:NormalizeRoot(a:root)
    if empty(l:root) || !s:EnsureSharedRoot(l:root)
        return ''
    endif
    return s:PrepareDir(l:root . '/' . a:user)
endfunction

" Delete backup and undo files older than s:prune_days days (0 disables).
" Age is measured by ctime, i.e. when the backup/undo file was created or last
" changed, so it isn't fooled by the original file's old mtime. Runs at most
" once per 24h (stamp file) and in the background when jobs are available, so
" it never delays startup. Swap files are never touched.
function! s:PruneCache(dir) abort
    if s:prune_days <= 0 || !executable('find')
        return
    endif
    let l:stamp = a:dir . '/.last_prune'
    if getftime(l:stamp) > localtime() - 86400
        return
    endif
    try
        call writefile([], l:stamp)
    catch
        return
    endtry
    let l:cmd = ['find', a:dir . '/backup', a:dir . '/undo', '-type', 'f',
                \ '-ctime', '+' . s:prune_days, '-delete']
    try
        if exists('*jobstart')
            " Neovim
            if jobstart(l:cmd, {'detach': v:true}) > 0
                return
            endif
        elseif has('job') && exists('*job_start')
            " Vim 8+; stoponexit='' so quitting right away doesn't kill it
            call job_start(l:cmd, {'in_io': 'null', 'out_io': 'null',
                        \ 'err_io': 'null', 'stoponexit': ''})
            return
        endif
    catch
        " fall through to the blocking path below
    endtry
    call system(join(map(copy(l:cmd), 'shellescape(v:val)'), ' ')
                \ . ' >/dev/null 2>&1')
endfunction

function! s:SetupCache() abort
    let $VIM_CACHE_DIR = ''
    let l:user = s:UserName()
    let l:dir  = ''

    " 1) Explicit override (per-user subdir under it)
    if !empty($UNISHELL_VIM_CACHE_DIR)
        let l:dir = s:PrepareUserDir($UNISHELL_VIM_CACHE_DIR, l:user)
        if empty(l:dir)
            call s:Warn('unusable UNISHELL_VIM_CACHE_DIR, falling back to '
                        \ . s:fallback_root)
        endif
    endif

    " 2) Fallback (per-user subdir under it)
    if empty(l:dir)
        let l:dir = s:PrepareUserDir(s:fallback_root, l:user)
    endif

    " 3) Give up: leave Vim's defaults alone (swap file next to the edited
    " file, no backup/undo files) rather than losing crash recovery
    if empty(l:dir)
        call s:Warn('no usable cache dir; using Vim defaults '
                    \ . '(swap files next to the edited file)')
        return
    endif

    let $VIM_CACHE_DIR = l:dir              " keep for other parts of your config

    " Only commas are special in these comma-separated options.
    " Trailing // => file names built from the full path (no collisions).
    let &directory = escape(l:dir . '/swap//',   ',')
    let &backupdir = escape(l:dir . '/backup//', ',')
    set swapfile backup

    if has('persistent_undo')
        let &undodir = escape(l:dir . '/undo//', ',')
        set undofile
    endif

    if has('viminfo')
        " Persist marks for 1000 files and 1000 history entries in a custom viminfo file.
        " 'n' must be last in 'viminfo'; the rest of the string is the filename.
        let &viminfo = "'1000,<1000,h,n" . l:dir . '/viminfo'
    endif

    call s:PruneCache(l:dir)
endfunction

call s:SetupCache()

" ┌────────────────┐
" │ netrw DIRECTORY│
" └────────────────┘
if !empty($VIM_CACHE_DIR)
    let g:netrw_home = $VIM_CACHE_DIR
endif

" ╚═══════════════════════════════════════════════════════════════



" ╔═══════════════════════════════════════════════════════════════
" ║ SOURCING VIM MODULAR CONFIGURATIONS

" Source files and/or directories located under an environment variable.
"
"   call s:SourceEnv('MY_CONF_DIR', 'options.vim', 'plugins.d', 'local')
"
" Each item is relative to $MY_CONF_DIR and can be:
"   - a file       -> sourced
"   - a directory  -> every *.vim file under it, recursively, in sorted order
" Unset variables and missing items are skipped silently.
"
" Deliberately NOT `abort`: an error inside one sourced file must not stop
" the remaining files from being sourced.
function! s:SourceEnv(var, ...)
    let l:base = substitute(eval('$' . a:var), '/\+$', '', '')
    if empty(l:base)
        return
    endif

    for l:rel in a:000
        let l:path = l:base . '/' . l:rel
        if isdirectory(l:path)
            " '/**/' also matches files directly inside the directory.
            " nosuf=1 so a user 'wildignore' can't silently hide files.
            let l:files = sort(glob(fnameescape(l:path) . '/**/*.vim', 1, 1))
        else
            let l:files = filereadable(l:path) ? [l:path] : []
        endif
        for l:f in l:files
            execute 'source' fnameescape(l:f)
        endfor
    endfor
endfunction

" EXAMPLE: call s:SourceEnv('UNISHELL_VIM_DIR', 'dir_to_source_all_.vim_files_under/', 'file1.vim', 'file2.vim')

" call s:SourceEnv('VIMRUNTIME', 'defaults.vim')    " NOT NEEDED | MOST DEFAULTS WILL BE LOADED BY THE FOLLOWING SOURCE(s)

" SOURCE PLUGINS
call s:SourceEnv('UNISHELL_VIM_CONFIG_PATH', '_plugins/_plugins.vim')

" SOURCE CONFIG MODULES
call s:SourceEnv('UNISHELL_VIM_CONFIG_PATH', 'config_modules')

" SOURCE FUNCTIONS
call s:SourceEnv('UNISHELL_VIM_CONFIG_PATH', '_functions') 


" ╚═══════════════════════════════════════════════════════════════



" ╔═══════════════════════════════════════════════════════════════
" ║ PROFILE-SPECIFIC SETTINGS

let s:profile = $UNISHELL_SESSION_SHELL_PROFILE

" ╭────────────────────────────────╮
" │ CLIPBOARD SETTINGS             │
" ╰────────────────────────────────╯
" Profile -> ordered list of clipboard tool candidates.
" To add a profile, add a line here; first executable candidate wins.
" Env vars are expanded; if one is unset, the literal '$VAR' path is
" simply not executable and gets skipped.
let s:profile_clip_tools = {
    \ 'work': ['$UNISHELL_SESSION_RUNTIME_DIR/scripts/rcopy/rcopy'],
    \ }

" Operators forwarded to the CLI tool. x, X, s and S (normal and visual
" modes) are kept out by the mappings below, which send them to the "-
" register (non-empty regname is skipped).
let s:clip_operators = ['y', 'd', 'c']

" Payloads above this many bytes use the blocking path in Vim, so we never
" rely on how a half-flushed pipe behaves when its input end is closed.
let s:clip_async_max = 32768

" Used for any profile not listed above (and no profile at all)
function! s:DefaultClipTool() abort
    if !empty($WAYLAND_DISPLAY) && executable('wl-copy')
        return 'wl-copy'
    endif
    return ''
endfunction

function! s:PickClipCmd() abort
    if has_key(s:profile_clip_tools, s:profile)
        for l:candidate in s:profile_clip_tools[s:profile]
            let l:path = expand(l:candidate)
            if executable(l:path)
                return l:path
            endif
        endfor
        " Profile is explicitly mapped but its tool is missing:
        " don't guess, fall through to native clipboard
        return ''
    endif
    return s:DefaultClipTool()
endfunction

" Feed <text> to the clipboard tool on stdin without blocking Vim.
" Newest yank wins: a previous job that is still running is stopped first.
" Output is discarded; the tool is started as a list, so no shell is involved.
function! s:SendToClip(text) abort
    if exists('*jobstart')
        " Neovim
        if exists('s:clip_job')
            silent! call jobstop(s:clip_job)
        endif
        let s:clip_job = jobstart([s:clip_cmd], {'detach': v:true})
        if s:clip_job > 0
            call chansend(s:clip_job, a:text)
            call chanclose(s:clip_job, 'stdin')
            return
        endif
    elseif has('job') && exists('*job_start') && strlen(a:text) <= s:clip_async_max
        " Vim 8+
        try
            if exists('s:clip_job') && job_status(s:clip_job) ==# 'run'
                call job_stop(s:clip_job)
            endif
            " stoponexit='' so `yy` followed by an immediate :q still completes
            let s:clip_job = job_start([s:clip_cmd], {
                        \ 'in_io': 'pipe', 'out_io': 'null', 'err_io': 'null',
                        \ 'stoponexit': ''})
            if job_status(s:clip_job) !=# 'fail'
                let l:ch = job_getchannel(s:clip_job)
                call ch_sendraw(l:ch, a:text)
                call ch_close_in(l:ch)
                return
            endif
        catch
            " fall through to the blocking path below
        endtry
    endif
    " No job support, job failed to start, or large payload: blocking but complete.
    " Redirect output so backgrounding tools (wl-copy) can't block system()
    call system(shellescape(s:clip_cmd) . ' >/dev/null 2>&1', a:text)
endfunction

" Send plain yanks/deletes/changes (no named register) to the CLI tool
function! s:YankToClip() abort
    if index(s:clip_operators, v:event.operator) < 0 || !empty(v:event.regname)
        return
    endif
    call s:SendToClip(@")
endfunction

let s:clip_cmd = s:PickClipCmd()

if !empty(s:clip_cmd) && exists('##TextYankPost')
    " Disable native clipboard, route manually via CLI
    set clipboard=
    augroup CustomClipboard
        autocmd!
        autocmd TextYankPost * call s:YankToClip()
    augroup END
    " x/X/s/S are reported as d/d/c/c; route them through "- so the handler
    " sees a register name and skips them (normal and visual mode)
    nnoremap x "-x
    nnoremap X "-X
    nnoremap s "-s
    nnoremap S "-S
    xnoremap x "-x
    xnoremap s "-s
    xnoremap S "-S
elseif has('clipboard')
    " Fallback: native clipboard (needs +clipboard / a provider in Neovim)
    set clipboard=unnamed
    if has('unnamedplus')
        set clipboard^=unnamedplus          " results in "unnamedplus,unnamed"
    endif
endif

" ╭────────────────────────────────╮
" │ PROFILE SOURCING               │
" ╰────────────────────────────────╯
if s:profile =~# '^[A-Za-z0-9_][A-Za-z0-9._-]*$'
    call s:SourceEnv('UNISHELL_CONFIG_PATH', 'vim/_profiles/' . s:profile . '.vim')
endif

" ╚═══════════════════════════════════════════════════════════════
