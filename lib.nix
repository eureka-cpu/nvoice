let
  lib = (import <nixpkgs> { }).lib;

  hexColor = lib.types.strMatching "^#[0-9a-fA-F]{6}$";

  # Option schema declarations — single source of truth for both validation
  # and help text generation. Adding or changing an option here automatically
  # updates both error messages and the output of `nix eval -f release.nix help`.
  #
  # Each entry: { path, type, required ? false, description, default ? <unset> }

  providerOptions = [
    { path = "name"; type = lib.types.str; required = true; description = "Your name shown in the invoice header"; }
    { path = "address"; type = lib.types.str; required = true; description = "Mailing address (use \\n for line breaks)"; }
    { path = "email"; type = lib.types.str; required = true; description = "Contact email"; }
  ];

  bankOptions = [
    { path = "name"; type = lib.types.str; required = true; description = "Payment method display name"; }
    { path = "note"; type = lib.types.str; required = false; description = "Optional note shown under the bank name"; }
  ];

  sectionOptions = [
    { path = "label"; type = lib.types.str; required = true; description = "Row label"; }
    { path = "value"; type = lib.types.str; required = true; description = "Row value"; }
    { path = "position"; type = lib.types.enum [ "left" "right" ]; required = false; description = "Column side"; default = "left"; }
  ];

  clientOptions = [
    { path = "address"; type = lib.types.str; required = false; description = "Client address printed on the invoice"; }
    { path = "billing"; type = lib.types.enum [ "hourly" "weekly" ]; required = false; description = "Billing mode"; default = "hourly"; }
  ];

  agencyOptions = [
    { path = "emblem"; type = lib.types.path; required = false; description = "Path to SVG shown in the invoice header"; }
    { path = "payment_terms"; type = lib.types.str; required = false; description = "Payment terms string"; default = "Net 30"; }
  ];

  colorKeys = [ "body" "label" "panel-left" "panel-right" "table-band" "rule" "dim" ];

  colorDescriptions = {
    body = "Main text";
    label = "Section and column labels";
    "panel-left" = "Left header panel background";
    "panel-right" = "Right header panel background (accent)";
    "table-band" = "Alternating table row fill";
    rule = "Borders and dividers";
    dim = "De-emphasised text (addresses, notes)";
  };

  fontOptions = [
    { path = "package"; type = lib.types.package; required = false; description = "nixpkgs font package (e.g. pkgs.inter)"; }
    { path = "family"; type = lib.types.str; required = false; description = "Font family name string (e.g. \"Inter\")"; }
  ];

  # Validate a list of option declarations against an attrset.
  # Returns a list of checked values (for use with builtins.deepSeq).
  checkOptions = prefix: opts: cfg:
    lib.concatLists (map
      (opt:
        let fullPath = "${prefix}.${opt.path}"; in
        if cfg ? ${opt.path} then [
          (if opt.type.check cfg.${opt.path}
          then cfg.${opt.path}
          else throw "config.nix: '${fullPath}' must be ${opt.type.description}")
        ] else if opt.required then
          throw "config.nix: '${fullPath}' is required"
        else [ ]
      )
      opts);

  validate = cfg:
    let
      banksType = lib.types.listOf lib.types.attrs;
    in
    builtins.deepSeq
      (checkOptions "provider" providerOptions cfg.provider
        ++ (if banksType.check cfg.banks then [ ]
      else throw "config.nix: 'banks' must be ${banksType.description}")
        ++ lib.concatLists (lib.imap0
        (i: bank:
          let
            sectionsType = lib.types.listOf lib.types.attrs;
            p = "banks[${toString i}]";
          in
          checkOptions p bankOptions bank
            ++ (if sectionsType.check bank.sections then [ ]
          else throw "config.nix: '${p}.sections' must be ${sectionsType.description}")
            ++ lib.concatLists (lib.imap0
            (j: section:
              checkOptions "${p}.sections[${toString j}]" sectionOptions section
            )
            bank.sections)
        )
        cfg.banks)
        ++ lib.optionals (cfg ? theme && cfg.theme ? colors)
        (lib.concatLists (map
          (key:
            lib.optionals (cfg.theme.colors ? ${key}) [
              (if hexColor.check cfg.theme.colors.${key} then cfg.theme.colors.${key}
              else throw "config.nix: 'theme.colors.${key}' must be ${hexColor.description}")
            ]
          )
          colorKeys))
        ++ lib.optionals (cfg ? theme && cfg.theme ? fonts && cfg.theme.fonts ? body)
        (checkOptions "theme.fonts.body" fontOptions cfg.theme.fonts.body)
        ++ lib.optionals (cfg ? clients)
        (lib.concatLists (lib.mapAttrsToList
          (name: c:
            checkOptions "clients.${name}" clientOptions c
          )
          cfg.clients))
        ++ lib.optionals (cfg ? agencies)
        (lib.concatLists (lib.mapAttrsToList
          (name: a:
            checkOptions "agencies.${name}" agencyOptions a
          )
          cfg.agencies)))
      cfg;

  esc = builtins.fromJSON ''"\u001B"'';
  blue = "${esc}[34m";
  bold = "${esc}[1m";
  dim = "${esc}[2m";
  reset = "${esc}[0m";

  # Format a list of option declarations into a help text block.
  fmtOptions = prefix: opts:
    lib.concatMapStringsSep "\n"
      (opt:
        let
          req = if opt.required then "required" else "optional";
          def = if opt ? default then ", default: ${toString opt.default}" else "";
          typ = opt.type.description;
        in
        "  ${blue}${bold}${prefix}.${opt.path}${reset}  ${dim}[${typ}, ${req}${def}]${reset}\n    ${opt.description}"
      )
      opts;

  # Exported schema for help text generation in release.nix.
  schema = {
    inherit providerOptions bankOptions sectionOptions clientOptions agencyOptions fontOptions colorKeys colorDescriptions;
    inherit fmtOptions hexColor blue bold dim reset;
  };

in
{ inherit validate schema; }
