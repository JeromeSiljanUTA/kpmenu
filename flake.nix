{
  description = "kpmenu - fast wofi frontend for keepass databases";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = {
    self,
    nixpkgs,
    ...
  }: let
    systems = ["x86_64-linux" "aarch64-linux"];
    forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
  in {
    packages = forAllSystems (pkgs: rec {
      kpmenu = pkgs.buildGoModule {
        pname = "kpmenu";
        version = "0.1.0";
        src = ./.;
        vendorHash = null;
        ldflags = ["-s" "-w"];
        meta = with pkgs.lib; {
          description = "Fast wofi frontend for keepass databases";
          license = licenses.gpl3Only;
          mainProgram = "kpmenu";
          platforms = platforms.linux;
        };
      };
      default = kpmenu;
    });

    overlays.default = final: prev: {
      kpmenu = self.packages.${final.system}.kpmenu;
    };

    # Import this in a home-manager / NixOS config. The database path is a real
    # option so each host sets it to wherever syncthing drops its vault.
    homeManagerModules.default = {
      config,
      lib,
      pkgs,
      ...
    }: let
      cfg = config.programs.kpmenu;
    in {
      options.programs.kpmenu = {
        enable = lib.mkEnableOption "kpmenu keepass menu";

        package = lib.mkOption {
          type = lib.types.package;
          default = self.packages.${pkgs.stdenv.hostPlatform.system}.kpmenu;
          description = "kpmenu package to use.";
        };

        database = lib.mkOption {
          type = lib.types.nullOr lib.types.str;
          default = null;
          example = "~/keepass/passwords.kdbx";
          description = "Path to the keepass database.";
        };

        keyFile = lib.mkOption {
          type = lib.types.nullOr lib.types.str;
          default = null;
          description = "Optional keyfile for the database.";
        };

        lockTimeout = lib.mkOption {
          type = lib.types.int;
          default = 3600;
          description = "Seconds of inactivity before the agent forgets the passphrase.";
        };

        clipboardClear = lib.mkOption {
          type = lib.types.int;
          default = 30;
          description = "Seconds before the copied password is cleared from the clipboard.";
        };

        keybinding = lib.mkOption {
          type = lib.types.nullOr lib.types.str;
          default = "Mod4+grave";
          description = "Sway/i3 keybinding bound to kpmenu, or null to skip.";
        };

        typeKeybinding = lib.mkOption {
          type = lib.types.nullOr lib.types.str;
          default = "Mod4+Shift+grave";
          description = "Sway/i3 keybinding bound to autotype, or null to skip.";
        };

        passphrasePrompt = lib.mkOption {
          type = lib.types.enum ["wofi" "pinentry"];
          default = "wofi";
          description = "How to prompt for the passphrase. wofi reuses the menu UI; pinentry needs a TTY or a GUI variant.";
        };

        pinentry = lib.mkOption {
          type = lib.types.package;
          default = pkgs.pinentry-gnome3;
          description = "Pinentry variant used when passphrasePrompt is pinentry.";
        };

        extraPackages = lib.mkOption {
          type = lib.types.listOf lib.types.package;
          default = [];
          description = "Extra runtime dependencies.";
        };
      };

      config = lib.mkIf cfg.enable (lib.mkMerge [
        {
          home.packages = [cfg.package cfg.pinentry]
            ++ (with pkgs; [keepassxc wtype wl-clipboard])
            ++ cfg.extraPackages;

          xdg.configFile."kpmenu/config".text = ''
            database = ${toString cfg.database}
            passphrase_prompt = ${cfg.passphrasePrompt}
            pinentry = ${lib.getExe cfg.pinentry}
          ''
          + lib.optionalString (cfg.keyFile != null) "keyfile = ${toString cfg.keyFile}\n"
          + ''
            lock_timeout = ${toString cfg.lockTimeout}
            clipboard_clear = ${toString cfg.clipboardClear}
          '';
        }

        (lib.mkIf (cfg.keybinding != null) {
          wayland.windowManager.sway.config.keybindings.${cfg.keybinding} = "exec ${cfg.package}/bin/kpmenu";
        })

        (lib.mkIf (cfg.typeKeybinding != null) {
          wayland.windowManager.sway.config.keybindings.${cfg.typeKeybinding} = "exec ${cfg.package}/bin/kpmenu type";
        })
      ]);
    };

    # Plain NixOS module (no home-manager) for completeness.
    nixosModules.default = {
      config,
      lib,
      pkgs,
      ...
    }: let
      cfg = config.programs.kpmenu;
    in {
      options.programs.kpmenu = {
        enable = lib.mkEnableOption "kpmenu keepass menu";
        package = lib.mkOption {
          type = lib.types.package;
          default = self.packages.${pkgs.stdenv.hostPlatform.system}.kpmenu;
          description = "kpmenu package to use.";
        };
        database = lib.mkOption {
          type = lib.types.nullOr lib.types.str;
          default = null;
          description = "Path to the keepass database.";
        };
        lockTimeout = lib.mkOption {
          type = lib.types.int;
          default = 3600;
          description = "Seconds of inactivity before the agent forgets the passphrase.";
        };
        passphrasePrompt = lib.mkOption {
          type = lib.types.enum ["wofi" "pinentry"];
          default = "wofi";
          description = "How to prompt for the passphrase.";
        };
      };

      config = lib.mkIf cfg.enable {
        environment.systemPackages = [cfg.package pkgs.keepassxc pkgs.wtype pkgs.wl-clipboard pkgs.pinentry-gnome3];
        environment.etc."kpmenu/config".text = ''
          database = ${toString cfg.database}
          passphrase_prompt = ${cfg.passphrasePrompt}
          lock_timeout = ${toString cfg.lockTimeout}
        '';
        environment.variables.KPMENU_CONFIG = "/etc/kpmenu/config";
      };
    };
  };
}
