# kpmenu

A fast, wofi-based frontend for KeePassXC databases. Pick an entry, get the
password in your clipboard or typed into the focused window.

## Why

Existing tools like keepmenu are excellent but are Python daemons that reparse
the database at launch, so the menu never feels instant. kpmenu keeps a flat
cache of entry paths on disk and only asks `keepassxc-cli` to decrypt the one
entry you selected. Launching the menu is just `read file` + `exec wofi`, so it
opens as fast as wofi itself.

The passphrase is held in a small resident agent's memory (not on disk) and is
forgotten after a configurable idle timeout. The agent is started on demand and
exits on its own when it times out.

## Requirements

- `keepassxc` (for `keepassxc-cli`)
- `wofi`
- `wl-clipboard` (for `wl-copy`/`wl-paste`)
- `wtype` (for autotype)
- a pinentry (defaults to `pinentry-curses`)

## Usage

```
kpmenu                pick an entry and copy its password
kpmenu type           pick an entry and autotype username + password
kpmenu copy [entry]   copy an entry without the menu
kpmenu refresh        rebuild the entry cache
kpmenu unlock         unlock the agent and warm the cache
kpmenu lock           forget the cached passphrase
```

Bind `kpmenu` and `kpmenu type` to keys in your compositor.

## Configuration

`~/.config/kpmenu/config` (INI-ish key = value):

```ini
database        = ~/keepass/passwords.kdbx
keyfile         = ~/keepass/key            # optional
pinentry        = pinentry-curses          # optional
keepassxc_cli   = keepassxc-cli
wofi            = wofi
wtype           = wtype
wl_copy         = wl-copy
clipboard_clear = 30                        # seconds; 0 disables
lock_timeout    = 3600                      # seconds of agent idle before lock
```

The `KPMENU_CONFIG` environment variable overrides the config path.

## Nix flake

The flake exposes the package plus home-manager and NixOS modules. The database
path is a real option, so each host points it at wherever its vault is synced:

```nix
{
  inputs.kpmenu.url = "git+http://git.nemesis.home.arpa/jerome/kpmenu";

  # home-manager
  imports = [ kpmenu.homeManagerModules.default ];
  programs.kpmenu = {
    enable = true;
    database = "~/keepass/passwords.kdbx";
    keyFile = "~/keepass/key";
  };
}
```

The home-manager module writes the config file, installs the runtime
dependencies, and (by default) binds `Mod4+grave` to open the menu and
`Mod4+Shift+grave` to autotype. Set `keybinding = null` / `typeKeybinding = null`
to manage bindings yourself.

Full option list: `database`, `keyFile`, `package`, `pinentry`, `lockTimeout`,
`clipboardClear`, `keybinding`, `typeKeybinding`, `extraPackages`.

## Design notes

- No comments in the source unless a decision is non-obvious.
- The cache is served immediately even when stale; a background agent call
  rebuilds it when the database mtime is newer. The menu never blocks on
  decryption.
- Group rows (trailing `/`) and `[empty]` markers from `keepassxc-cli ls -R`
  are filtered out.
