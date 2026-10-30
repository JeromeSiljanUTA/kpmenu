# kpmenu

** THIS PROJECT WAS FULLY VIBE CODED **:w

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
kpmenu view           browse, read, edit and add entries interactively
kpmenu copy [entry]   copy an entry without the menu
kpmenu refresh        rebuild the entry cache
kpmenu unlock         unlock the agent and warm the cache
kpmenu lock           forget the cached passphrase
```

`kpmenu view` prompts for the passphrase up front, then lets you drill into an
entry. The field menu lists every field `keepassxc-cli show --all` reports,
including custom attributes. Selecting a field copies it to the clipboard and
autotypes it.

- **Notes** open a line picker: choose a line to copy and autotype it, or `Edit`
  to open the whole notes blob in `$EDITOR` (launched in `terminal`).
- **Edit** changes Title, UserName, Password, URL or Notes, or deletes the
  entry. Password offers typed or generated values; generated passwords are
  also placed on the clipboard.
- **Add entry** creates an entry in a chosen group with a typed or generated
  password.

Custom attributes can be read, copied and typed. `keepassxc-cli` has no way to
write custom attributes, so edit those in the KeePassXC GUI.

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
hide_groups     = Recycle Bin               # comma separated, hidden from the menu
terminal        = foot                      # terminal used to run the editor
editor          = nvim                      # defaults to $EDITOR, then nvim
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

The home-manager module binds `keybinding` (copy), `typeKeybinding`
(autotype) and `viewKeybinding` (interactive view). Defaults are `Mod4+grave`,
`Mod4+Shift+grave` and `Mod4+Shift+Return`; the interactive view is the
recommended one, exposed as `Mod4+v` in the STARSCREAM config. Set any to
`null` to skip it.

Full option list: `database`, `keyFile`, `package`, `pinentry`,
`passphrasePrompt`, `lockTimeout`, `clipboardClear`, `keybinding`,
`typeKeybinding`, `viewKeybinding`, `extraPackages`.

## Design notes

- No comments in the source unless a decision is non-obvious.
- The cache is served immediately even when stale; a background agent call
  rebuilds it when the database mtime is newer. The menu never blocks on
  decryption.
- Group rows (trailing `/`) and `[empty]` markers from `keepassxc-cli ls -R`
  are filtered out, along with any group listed in `hide_groups`.
- kpmenu is a porcelain over `keepassxc-cli`, not a reimplementation. All
  parsing, decryption and writing is delegated to it, so no password-handling
  crypto lives in kpmenu.
- The agent runs `keepassxc-cli export` once per unlock, parsing the whole
  database into memory. Listing entries, opening an entry and reading any of
  its fields then cost no further decrypts, so the detail screen is instant
  even on a large vault. The cache refreshes when the file changes or an edit
  is written. TOTP is fetched lazily because it changes over time. Escape
  moves back one menu; escape on the entry list exits.
- `keepassxc-cli show --all` lists custom attributes, but `edit` cannot write
  them, so custom attributes are read-only in kpmenu.
