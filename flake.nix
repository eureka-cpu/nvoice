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

          logHours = pkgs.writeShellApplication {
            name = "log-hours";
            runtimeInputs = with pkgs; [ jq coreutils getent ];
            text = ''
              usage() {
                cat <<'EOF'
              Usage: log-hours [DATE] -hours N [options]
                DATE  YYYYMMDD or YYYY-MM-DD (default: today)

              Options:
                -hours N               hours worked (required)
                -client NAME           client name
                -agency NAME           agency name
                -task TEXT             task description
                -user NAME             worker name
                -rate N                hourly rate in source currency
                -currency SYM          source currency
                -target-currency SYM   target currency (default: same as -currency)
                -exchange-rate N       exchange rate (default: 1.0)
                -file PATH             output file (default: <monthname>.json)
              EOF
                exit 1
              }

              DATE=""
              HOURS=""
              CLIENT=""
              AGENCY=""
              TASK=""
              USER_NAME=""
              RATE=""
              CURRENCY=""
              TARGET_CURRENCY=""
              EXCHANGE_RATE="1.0"
              FILE=""

              if [[ $# -gt 0 && "$1" != -* ]]; then
                DATE="$1"
                shift
              fi

              while [[ $# -gt 0 ]]; do
                case "$1" in
                  -date)            DATE="$2";            shift 2 ;;
                  -hours)           HOURS="$2";           shift 2 ;;
                  -client)          CLIENT="$2";          shift 2 ;;
                  -agency)          AGENCY="$2";          shift 2 ;;
                  -task)            TASK="$2";            shift 2 ;;
                  -user)            USER_NAME="$2";       shift 2 ;;
                  -rate)            RATE="$2";            shift 2 ;;
                  -currency)        CURRENCY="$2";        shift 2 ;;
                  -target-currency) TARGET_CURRENCY="$2"; shift 2 ;;
                  -exchange-rate)   EXCHANGE_RATE="$2";   shift 2 ;;
                  -file)            FILE="$2";            shift 2 ;;
                  *) echo "unknown option: $1"; usage ;;
                esac
              done

              [[ -z "$DATE" ]] && DATE=$(date +%Y%m%d)
              [[ -z "$HOURS" ]] && { echo "error: -hours is required"; usage; }

              if [[ -z "$USER_NAME" ]]; then
                USER_NAME=$(getent passwd "$(id -un)" | cut -d: -f5 | cut -d, -f1)
              fi

              DATE="''${DATE//-/}"

              if [[ -z "$FILE" ]]; then
                MONTH_NAME=$(date -d "''${DATE:0:4}-''${DATE:4:2}-01" +%B | tr '[:upper:]' '[:lower:]')
                FILE="''${MONTH_NAME}.json"
              fi

              TARGET_CURRENCY="''${TARGET_CURRENCY:-$CURRENCY}"

              if [[ -n "$RATE" ]]; then
                TARGET_RATE=$(jq -n "$RATE * $EXCHANGE_RATE")
                SOURCE_COST=$(jq -n "$HOURS * $RATE")
                TARGET_COST=$(jq -n "$HOURS * $TARGET_RATE")
              else
                TARGET_RATE="null"
                SOURCE_COST="null"
                TARGET_COST="null"
              fi

              [[ ! -f "$FILE" ]] && echo "[]" > "$FILE"

              jq \
                --arg     agency          "$AGENCY" \
                --arg     client          "$CLIENT" \
                --arg     date            "$DATE" \
                --argjson exchange_rate   "$EXCHANGE_RATE" \
                --argjson hours           "$HOURS" \
                --argjson source_cost     "''${SOURCE_COST:-null}" \
                --arg     source_currency "$CURRENCY" \
                --argjson source_rate     "''${RATE:-null}" \
                --argjson target_cost     "''${TARGET_COST:-null}" \
                --arg     target_currency "$TARGET_CURRENCY" \
                --argjson target_rate     "''${TARGET_RATE:-null}" \
                --arg     task            "$TASK" \
                --arg     user            "$USER_NAME" \
                '. + [{
                  agency:             $agency,
                  client:             $client,
                  start_date:         $date,
                  end_date:           $date,
                  exchange_rate:      $exchange_rate,
                  rounded_hours:      $hours,
                  source_cost:        $source_cost,
                  source_currency:    $source_currency,
                  source_hourly_rate: $source_rate,
                  target_cost:        $target_cost,
                  target_currency:    $target_currency,
                  target_hourly_rate: $target_rate,
                  task:               $task,
                  user:               $user
                } | with_entries(select(.value != null and .value != ""))]' \
                "$FILE" > "$FILE.tmp" && mv "$FILE.tmp" "$FILE"

              echo "Logged ''${HOURS}h on ''${DATE} → ''${FILE}"
            '';
          };

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
          log-hours = {
            type = "app";
            program = "${logHours}/bin/log-hours";
          };
        }
      );
    };
}
