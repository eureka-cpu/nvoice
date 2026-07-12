# Invoice generator.
#
# Pass entries as a path or JSON string. Arrays are treated as harvest-exporter
# output and produce one PDF per client. A single object builds one invoice.
#
#   nix-build --arg entries ./entries.json
#   nix-build --argstr entries "$(harvest-exporter --year 2026 --month 1 --format json)"
#   nix-build --argstr invoiceNumber INV-2026-01

{ entries ? builtins.readFile ./entries-example.json
, org ? ""
, date ? null
, invoiceNumber ? null
, emblem ? null
  # combine = true  → one invoice per client (all their entries as line items)
  # combine = false → one invoice per entry
, combine ? false
}:
let
  inherit (import ./release.nix) pkgs lib clients agencies weekOfYear buildInvoice;

  entriesStr = if builtins.typeOf entries == "path" then builtins.readFile entries else entries;
  parsed = builtins.fromJSON entriesStr;

  makeInvoiceData =
    idx: clientName: clientEntries:
    let
      first = builtins.head clientEntries;
      agCfg = agencies.${first.agency} or { };
      clientCfg = clients.${clientName} or { };
      billing = clientCfg.billing or "hourly";
      hasFx = first.exchange_rate != 1.0;
      number = "INV-${builtins.substring 0 4 first.end_date}-${lib.fixedWidthString 2 "0" (toString (idx + 1))}";
    in
    {
      inherit number billing;
      client = { name = clientName; } // lib.optionalAttrs (clientCfg ? address) { inherit (clientCfg) address; };
      currency = first.source_currency;
      payment_terms = agCfg.payment_terms or "Net 30";
      entries = map
        (
          e:
          { description = e.task; }
          // (
            if billing == "weekly"
            then { week = weekOfYear e.start_date; amount = e.source_cost; }
            else { hours = e.rounded_hours; rate = e.source_hourly_rate; }
          )
        )
        clientEntries;
    }
    // lib.optionalAttrs hasFx {
      currency_to = first.target_currency;
      exchange_rate = first.exchange_rate;
    };

  clientSafeStr = name: lib.pipe name [
    (lib.replaceStrings [ "." "," ] [ "" "" ])
    (lib.replaceStrings [ " " "/" ] [ "_" "_" ])
    (lib.removeSuffix "_")
  ];

  invoiceFiles = invoiceData: drv: filename:
    [
      { name = "${filename}.pdf"; path = "${drv}/${filename}.pdf"; }
      { name = "${filename}.typ"; path = "${drv}/${filename}.typ"; }
    ];

  harvestEntries = if builtins.isList parsed then parsed else [ parsed ];

  # One invoice per client, all their entries as line items
  batchBuild =
    let
      byClient = lib.groupBy (e: e.client) harvestEntries;
      sortedClients = builtins.sort (a: b: a < b) (builtins.attrNames byClient);
      invoiceLinks = lib.imap0
        (
          idx: clientName:
            let
              clientEntries = byClient.${clientName};
              first = builtins.head clientEntries;
              invoiceData = makeInvoiceData idx clientName clientEntries;
              datePart = "${builtins.substring 0 4 first.end_date}-${builtins.substring 4 2 first.end_date}";
              effectiveNumber = if invoiceNumber != null then toString invoiceNumber else invoiceData.number;
              filename = "${clientSafeStr clientName}_${effectiveNumber}_${datePart}";
              drv = buildInvoice {
                inherit filename invoiceNumber;
                emblem = (agencies.${first.agency} or { }).emblem or null;
                entries = builtins.toJSON invoiceData;
              };
            in
            invoiceFiles invoiceData drv filename
        )
        sortedClients;
    in
    pkgs.linkFarm "invoices" (lib.flatten invoiceLinks);

  # One invoice per entry
  separateBuild =
    let
      invoiceLinks = lib.imap0
        (
          idx: e:
            let
              invoiceData = makeInvoiceData idx e.client [ e ];
              datePart = "${builtins.substring 0 4 e.end_date}-${builtins.substring 4 2 e.end_date}";
              effectiveNumber = if invoiceNumber != null then toString invoiceNumber else invoiceData.number;
              filename = "${clientSafeStr e.client}_${effectiveNumber}_${datePart}";
              drv = buildInvoice {
                inherit filename invoiceNumber;
                emblem = (agencies.${e.agency} or { }).emblem or null;
                entries = builtins.toJSON invoiceData;
              };
            in
            invoiceFiles invoiceData drv filename
        )
        harvestEntries;
    in
    pkgs.linkFarm "invoices" (lib.flatten invoiceLinks);

in
if builtins.isList parsed || (parsed ? agency) then
  if combine then batchBuild else separateBuild
else
  buildInvoice { entries = entriesStr; inherit emblem org date invoiceNumber; }
