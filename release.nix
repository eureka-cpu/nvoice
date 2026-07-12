{ configFile ? null }:
let
  pkgs = import <nixpkgs> { };
  inherit (pkgs)
    lib
    stdenvNoCC
    typst
    typstyle
    ;

  nvoiceLib = import ./lib.nix;
  inherit (nvoiceLib) schema;

  # Private: loaded without validation, used only for devshell / clients / agencies
  _defaultConfigFile =
    if configFile != null then configFile
    else if builtins.pathExists ./config.nix then ./config.nix
    else ./config-example.nix;
  _rawConfig = import _defaultConfigFile { inherit pkgs; };

  _userFontPackages = lib.mapAttrsToList (_: f: f.package) ((_rawConfig.theme or { }).fonts or { });
  _fontPkgs = _userFontPackages;
  _TYPST_FONT_PATHS = lib.optionalString (_fontPkgs != [ ])
    (lib.concatMapStringsSep ":" (p: "${p}/share/fonts") _fontPkgs);

  toConfigJson =
    cfg:
    let
      theme = cfg.theme or { };
      fonts = theme.fonts or { };
      stripped = theme // lib.optionalAttrs (fonts != { }) { fonts = lib.mapAttrs (_: f: f.family) fonts; };
    in
    builtins.toJSON (if cfg ? theme then cfg // { theme = stripped; } else cfg);

  configJson = toConfigJson _rawConfig;

  clients = _rawConfig.clients or { };
  agencies = _rawConfig.agencies or { };

  weekOfYear =
    dateStr:
    let
      fromDateStr =
        ds:
        let
          month = lib.toInt (lib.removePrefix "0" (builtins.substring 4 2 ds));
          day = lib.toInt (lib.removePrefix "0" (builtins.substring 6 2 ds));
          acc = [ 0 31 59 90 120 151 181 212 243 273 304 334 ];
          dayOfYear = builtins.elemAt acc (month - 1) + day;
          week = builtins.div (dayOfYear - 1) 7 + 1;
        in
        if week > 52 then 52 else week;
      fromCurrentTime =
        let
          totalDays = builtins.div builtins.currentTime 86400;
          isLeap = y: (lib.mod y 4 == 0 && lib.mod y 100 != 0) || lib.mod y 400 == 0;
          daysInYear = y: if isLeap y then 366 else 365;
          yearResult = lib.foldl'
            (
              acc: y:
                if acc.done || acc.remaining < (daysInYear y)
                then acc // { done = true; year = y; }
                else acc // { remaining = acc.remaining - (daysInYear y); }
            )
            { remaining = totalDays; done = false; year = 1970; }
            (lib.range 1970 2200);
          dayOfYear = yearResult.remaining + 1;
          week = builtins.div (dayOfYear - 1) 7 + 1;
        in
        if week > 52 then 52 else week;
    in
    if dateStr == null then fromCurrentTime else fromDateStr dateStr;

  buildInvoice =
    { entries
    , configFile ? _defaultConfigFile
    , emblem ? null
    , org ? ""
    , date ? null
    , invoiceNumber ? null
    , filename ? "invoice"
    }:
    let
      rawConfig = nvoiceLib.validate (import configFile { inherit pkgs; });
      userFontPackages = lib.mapAttrsToList (_: f: f.package) ((rawConfig.theme or { }).fonts or { });
      fontPkgs = userFontPackages;
      TYPST_FONT_PATHS = lib.optionalString (fontPkgs != [ ])
        (lib.concatMapStringsSep ":" (p: "${p}/share/fonts") fontPkgs);
      builtConfigJson = toConfigJson rawConfig;
      resolvedEmblem =
        if emblem != null then emblem
        else if builtins.pathExists ./emblem.svg then ./emblem.svg
        else null;
      hasEmblem = resolvedEmblem != null;
    in
    stdenvNoCC.mkDerivation (
      {
        name = "invoice";
        src = lib.cleanSourceWith {
          src = lib.cleanSource ./.;
          filter =
            path: _type:
            !(lib.hasSuffix ".nix" path)
            && !(lib.hasSuffix ".pdf" path)
            && !(lib.hasSuffix ".schema.json" path)
            && !(lib.hasSuffix "entries.json" path);
        };

        SOURCE_DATE_EPOCH = toString builtins.currentTime;

        configJson = builtConfigJson;
        inherit entries org filename;
        emblemSrc = if resolvedEmblem != null then "${resolvedEmblem}" else "";
        dateArg = if date != null then date else "";
        invoiceNumberArg = if invoiceNumber != null then toString invoiceNumber else "";

        buildInputs = fontPkgs;
        nativeBuildInputs = [ typst ];

        doCheck = true;
        nativeCheckInputs = [ typstyle ];
        checkPhase = ''
          typstyle --check src/invoice.typ
        '';

        buildPhase = ''
          if [ -z "$entries" ]; then
            echo "error: no invoice entries provided" >&2
            echo "  pass --arg entries ./entries.json" >&2
            exit 1
          fi

          if [ -n "$dateArg" ]; then
            export SOURCE_DATE_EPOCH=$(date -d "$dateArg" +%s)
          fi

          printf '%s' "$configJson" > config.json
          printf '%s' "$entries" > entries.json

          if [ -n "$emblemSrc" ]; then
            cp "$emblemSrc" emblem.svg
          fi

          extra=""
          if [ -n "$invoiceNumberArg" ]; then
            extra="--input invoice-number=$invoiceNumberArg"
          fi

          typst compile src/invoice.typ "$filename.pdf" \
            --root . \
            --input org="$org" \
            ${lib.optionalString hasEmblem "--input has-emblem=true"} \
            $extra
        '';

        installPhase = ''
          mkdir -p $out
          cp "$filename.pdf" $out/
          cp src/invoice.typ "$out/$filename.typ"
        '';
      }
      // lib.optionalAttrs (fontPkgs != [ ]) { inherit TYPST_FONT_PATHS; }
    );

  # Dynamic config reference derived entirely from the schema declarations in lib.nix.
  # If an option is added, renamed, or its type changed there, this updates automatically.
  configOptions =
    let
      inherit (schema) blue bold dim reset hexColor colorKeys colorDescriptions fmtOptions;
      hdr = title: "${bold}${title}${reset}";
      colorHelp = lib.concatMapStringsSep "\n"
        (key:
          "  ${blue}${bold}theme.colors.${key}${reset}  ${dim}[${hexColor.description}, optional]${reset}\n    ${colorDescriptions.${key}}"
        )
        colorKeys;
    in
    ''
        ${hdr "provider  (required)"}
      ${fmtOptions "provider" schema.providerOptions}

        ${hdr "banks[]  (required, one or more)"}
      ${fmtOptions "banks[]" schema.bankOptions}

        ${hdr "banks[].sections[]  (required)"}
      ${fmtOptions "banks[].sections[]" schema.sectionOptions}

        ${hdr "clients.<name>  (optional)"}
      ${fmtOptions "clients.<name>" schema.clientOptions}

        ${hdr "agencies.<name>  (optional)"}
      ${fmtOptions "agencies.<name>" schema.agencyOptions}

        ${hdr "theme.colors  (optional, all six-digit hex)"}
      ${colorHelp}

        ${hdr "theme.fonts.body  (optional)"}
      ${fmtOptions "theme.fonts.body" schema.fontOptions}
    '';

  devshell = pkgs.mkShell (
    {
      buildInputs = _fontPkgs;
      packages = with pkgs; [
        typst
        typstyle
        nil
        nixpkgs-fmt
        go
      ];
    }
    // lib.optionalAttrs (_fontPkgs != [ ]) { TYPST_FONT_PATHS = _TYPST_FONT_PATHS; }
  );

in
{
  inherit
    pkgs
    lib
    configJson
    configOptions
    clients
    agencies
    weekOfYear
    buildInvoice
    devshell
    ;
}
