{
  description = "nvoice — Nix invoice generator using Typst";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      release = import ./release.nix;
    in
    {
      # Use as a lib in another flake:
      #
      #   inputs.nvoice.url = "github:you/nvoice";
      #
      #   nvoice.lib.buildInvoice {
      #     entries = builtins.toJSON { ... };
      #   }
      lib = {
        inherit (release) buildInvoice weekOfYear configOptions;
      };

      devShells = forAllSystems (_: {
        default = release.devshell;
      });

      # nix run . -- entries.json [OPTIONS]
      apps = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          nvoice = pkgs.writeShellApplication {
            name = "nvoice";
            runtimeInputs = [ pkgs.nix ];
            text = ''
              usage() {
                local show_config="''${1:-false}"
                echo "usage: nvoice [OPTIONS] [ENTRIES_FILE]"
                echo ""
                echo "  ENTRIES_FILE            harvest JSON array or single invoice object"
                echo "                          (default: ./entries.json)"
                echo ""
                echo "  OPTIONS"
                echo "  --combine               one invoice per client (default: one per entry)"
                echo "  --org <name>            override provider name (single-invoice mode)"
                echo "  --date <YYYY-MM-DD>     invoice date (single-invoice mode)"
                echo "  --invoice-number <id>   override invoice number (single-invoice mode)"
                echo "  --open                  open result directory after building"
                echo ""
                echo "  --help -h               show this help message (long help includes config options)"
                if [[ "$show_config" == "true" ]]; then
                  echo ""
                  echo "  CONFIG"
                  nix eval --raw -f release.nix configOptions
                fi
              }

              entries=""
              combine=false
              org=""
              date=""
              invoice_number=""
              open_result=false

              while [[ $# -gt 0 ]]; do
                case $1 in
                  --combine)
                    combine=true;
                    shift
                    ;;
                  --org)
                    org="$2";
                    shift 2
                    ;;
                  --date)
                    date="$2";
                    shift 2
                    ;;
                  --invoice-number)
                    invoice_number="$2";
                    shift 2
                    ;;
                  --open)
                    open_result=true;
                    shift
                    ;;
                  -h)
                    usage false;
                    exit 0
                    ;;
                  --help)
                    usage true;
                    exit 0
                    ;;
                  -*)
                    echo "unknown flag: $1" >&2;
                    usage >&2;
                    exit 1
                    ;;
                  *)
                    entries="$1";
                    shift
                    ;;
                esac
              done

              if [[ -z "$entries" ]]; then
                if [[ -f ./entries.json ]]; then
                  entries="./entries.json"
                else
                  echo "error: no entries file specified and ./entries.json not found" >&2
                  usage >&2
                  exit 1
                fi
              fi

              args=()
              [[ "$combine" == true ]] && args+=(--arg combine true)
              [[ -n "$org" ]] && args+=(--argstr org "$org")
              [[ -n "$date" ]] && args+=(--argstr date "$date")
              [[ -n "$invoice_number" ]] && args+=(--argstr invoiceNumber "$invoice_number")

              result=$(nix-build ${self}/default.nix \
                --argstr entries "$(cat "$entries")" \
                "''${args[@]}")

              echo "$result"
              if [[ "$open_result" == true ]]; then
                for pdf in "$result"/*.pdf; do
                  ${if pkgs.stdenv.isDarwin then "open" else "xdg-open"} "$pdf"
                done
              fi
            '';
          };
        in
        {
          default = {
            type = "app";
            program = "${nvoice}/bin/nvoice";
          };
        }
      );
    };
}
