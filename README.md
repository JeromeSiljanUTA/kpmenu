# kpmenu

**THIS PROJECT WAS FULLY VIBE CODED**

A fast, wofi-based wrapper for `keepassxc-cli`.

## Why

I wanted something super fast.

## Requirements

- `keepassxc` (for `keepassxc-cli`)
- `wofi`
- `wl-clipboard` (for `wl-copy`/`wl-paste`)
- `wtype` (for autotype)
- a terminal + `$EDITOR` (for editing notes; default terminal is `foot`)
- a pinentry, if you set `passphrase_prompt = pinentry`

## Usage

```
kpmenu                pick an entry, copy its password
kpmenu type           pick an entry, autotype username + password
kpmenu view           browse, read, edit and add entries interactively
kpmenu copy [entry]   copy an entry's password without the menu
kpmenu refresh        rebuild the entry cache
kpmenu unlock         unlock the agent and warm the cache
kpmenu lock           forget the cached passphrase
```

`view` prompts for the passphrase, then lists entries (last one first, as `↻ entry`).
Picking a field copies and types it and closes the menu; `Edit ▸` opens
change/delete actions and `Add entry` creates new entries. Esc goes back one
menu. Ctrl-n/Ctrl-p navigate.

## Config

`~/.config/kpmenu/config` (override path with `$KPMENU_CONFIG`):

```ini
database          = ~/keepass/Database.kdbx   # required
keyfile           = ~/keepass/key             # optional
passphrase_prompt = wofi                       # wofi or pinentry
terminal          = foot                       # terminal for the notes editor
editor            = nvim                       # defaults to $EDITOR, then nvim
clipboard_clear   = 30                         # seconds; 0 disables
lock_timeout      = 3600                       # seconds before the agent locks
hide_groups       = Recycle Bin                # comma separated
```

`keepassxc_cli`, `wofi`, `wtype`, `wl_copy`, `pinentry` and `cache_dir` can also
be set. The flake's home-manager module writes this file; the database path is a
`programs.kpmenu.database` option.
