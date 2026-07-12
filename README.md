# Nvoice

A Nix-based invoice generator using Typst. Configure your identity and bank
details once in `config.nix`, then pass harvest-exporter JSON to generate one
PDF per client automatically.

## Quick Start

```sh
# With flakes
nix run . -- entries.json
nix run . -- entries.json --combine --open

# Without flakes
nix-build --arg entries ./entries.json

open result/
```

Pass `entries` as a file path (`--arg`) or an inline JSON string (`--argstr`).

- **Array** (harvest-exporter output) → one PDF per entry or per client depending on `combine`
- **Single object** (internal schema) → one PDF

See `entries-example.json` for the harvest format and `entries.schema.json` for
the full field reference.

## Tracking Hours Without a Time Service

If you don't use Harvest or a similar service, you can log hours daily with the
included `log-hours` script. It appends one entry per day to a per-month JSON
file (e.g. `july.json`) and derives the filename automatically from the date.

Run it directly with the flake — `jq` and `coreutils` are bundled automatically.
DATE defaults to today if omitted:

```sh
nix run .#log-hours -- 2026-07-14 \
  -hours 8 \
  -client "Acme Corp" \
  -agency "Example Agency LLC" \
  -rate 150 \
  -currency '$' \
  -task "Backend Development" \
  -user "Jane Smith"
```

All flags are optional — omit any you don't need. `source_cost` and
`target_cost` are calculated from `-rate` and `-hours` automatically. For
invoices billed in a single currency, `-target-currency` and `-exchange-rate`
can be omitted.

```sh
# With currency conversion
nix run .#log-hours -- 2026-07-14 -hours 8 -rate 150 -currency '$' \
  -target-currency '€' -exchange-rate 0.92

# Write to a specific file instead of the default <month>.json
nix run .#log-hours -- 2026-07-14 -hours 8 -file q3.json

# YYYYMMDD date format also accepted
nix run .#log-hours -- 20260714 -hours 8 -rate 150 -currency USDC
```

At the end of the month, pass the file to generate the invoice:

```sh
nix-build --arg entries ./july.json --argstr invoiceNumber 2026-07 --combine
```

Each day becomes its own line item. The `--combine` flag merges all entries for
the same client into one invoice. Month files are git-ignored by default — add
them to a private repo or keep them local.

## Using as a Library

To generate invoices as part of another Nix project:

```nix
# flake.nix
inputs.nvoice.url = "github:you/nvoice";

outputs = { self, nvoice, ... }: {
  packages.x86_64-linux.invoice = nvoice.lib.buildInvoice {
    entries = builtins.toJSON { ... };
  };
};
```

## Introspection

```sh
# All config.nix options with types, required/optional, and defaults
nix eval --raw -f release.nix configOptions

# Currently active config, pretty-printed with colour
nix eval --raw -f release.nix configJson | jq '.'
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--arg entries ./path.json` | `entries-example.json` | Harvest JSON array or single invoice object |
| `--arg combine true` | `false` | `true` = one invoice per client; `false` = one invoice per entry |
| `--argstr org "Name"` | `""` | Override the provider name on a single invoice |
| `--argstr date "2026-01-31"` | current date | Invoice date (single invoice mode only) |
| `--argstr invoiceNumber INV-01` | from entries | Override the invoice number |

> [!WARNING]
> Use `--argstr` (not `--arg`) for all string values such as `invoiceNumber`,
> `org`, and `date`. `--arg` evaluates its value as a Nix expression, so
> `--arg invoiceNumber 2026-06` computes `2026 − 6 = 2020` instead of passing
> the string `"2026-06"`. `entries` and `combine` are the only flags that
> legitimately use `--arg`.

## Config (`config.nix`)

Copy `config-example.nix` to `config.nix` and fill in your details. If
`config.nix` is absent, `config-example.nix` is used automatically.

```nix
{ pkgs ? import <nixpkgs> { } }:
{
  # Required: your identity shown in the invoice header
  provider = {
    name    = "Jane Smith";
    address = "123 Main St\nNew York, NY 10001";
    email   = "jane@example.com";
  };

  # Required: one or more payment method blocks
  banks = [
    {
      name = "Example Bank ACH";
      note = "Preferred for domestic transfers";  # optional
      sections = [
        { label = "Account Holder";        value = "Jane Smith"; }
        { label = "Bank name and address"; value = "Example Bank\n..."; }
        { label = "Account Number"; value = "123456789"; position = "right"; }
        { label = "Routing Number"; value = "021000021"; position = "right"; }
      ];
    }
  ];

  # Optional: per-client overrides (matched against harvest client field)
  clients = {
    "Acme Corp" = {
      address = "123 Main St\nNew York, NY 10001";  # optional
      billing = "weekly";  # "hourly" (default) or "weekly"
    };
  };

  # Optional: per-agency settings (matched against harvest agency field)
  agencies = {
    "Acme Corp" = {
      emblem        = ./emblem-example.svg;  # SVG shown in the invoice header
      payment_terms = "Net 30";              # default: "Net 30"
    };
  };

  # Optional: visual customisation
  theme.colors = {
    body        = "#464646";  # main text
    label       = "#323232";  # section/column labels
    panel-left  = "#f3f3f3";  # left header panel background
    panel-right = "#6d98c2";  # right header panel background (accent)
    table-band  = "#e8e8e8";  # alternating table row fill
    rule        = "#d2d2d2";  # borders and dividers
    dim         = "#949494";  # de-emphasised text (addresses, notes)
  };

  # Optional: custom font (any nixpkgs font package)
  theme.fonts.body = {
    package = pkgs.inter;
    family  = "Inter";
  };
}
```

Colors must be six-digit hex strings (`#rrggbb`). `section.position` is either
`"left"` (default) or `"right"`.

## Agencies and Emblems

The `agency` field in the harvest output is matched against the `agencies`
section in `config.nix`. An emblem SVG and payment terms can be set per agency.
If an agency has no entry or no `emblem` key, no emblem is shown.

## Validating Your Config

```sh
nix-instantiate --eval --strict -E \
  '(import ./lib.nix).validate (import ./config.nix { })'
```

A valid config prints the evaluated attrset. An invalid config throws a
descriptive error, for example:

```
error: config.nix: 'theme.colors.body' must be string matching the pattern ^#[0-9a-fA-F]{6}$
```

## Developing with Typst Directly

For live preview with `typst watch`:

```sh
# Enter devshell (sets TYPST_FONT_PATHS automatically)
nix develop          # with flakes
nix-shell -A devshell release.nix  # without flakes

# Generate config.json (required once, and after any config.nix change)
nix eval --raw -f release.nix configJson > config.json

# Create entries.json in the internal schema (what Typst reads directly)
cat > entries.json << 'EOF'
{
  "client": { "name": "Acme Corp", "address": "123 Main St\nNew York, NY 10001" },
  "number": "INV-2026-001",
  "currency": "$",
  "payment_terms": "Net 30",
  "billing": "hourly",
  "entries": [
    { "description": "Backend Development", "hours": 24, "rate": 150 },
    { "description": "Code Review",          "hours":  8, "rate": 150 }
  ]
}
EOF

# Compile or watch
typst compile src/invoice.typ --root .
typst watch src/invoice.typ --root .
```

> [!NOTE]
> `--root .` is required because the template reads `config.json` and
> `entries.json` from the project root. `config.json` is the Nix-serialised
> form of `config.nix` and should not be edited directly.

Each build also writes `<filename>.typ` alongside the PDF in `result/`, so you
can inspect or manually adjust a compiled invoice after the fact.
