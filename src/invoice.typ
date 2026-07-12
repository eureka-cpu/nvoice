// Values come from three places:
//   config.nix   — provider identity and bank accounts (edit once)
//   entries.json — invoice metadata and timesheet lines (edit per invoice)
//   --input flags — org (overrides client name), invoice-number (set via nix-build args)

#let cfg = json("../config.json")
#let data = json("../entries.json")

#let client = data.client
#let client-name = {
  let o = sys.inputs.at("org", default: "")
  if o == "" { client.name } else { o }
}
#let client-address = client.at("address", default: "")

#let invoice-number = {
  let n = sys.inputs.at("invoice-number", default: "")
  if n == "" { data.number } else { n }
}

#let provider = cfg.provider
#let banks = cfg.banks
#let currency = data.currency
#let currency-to = data.at("currency_to", default: none)
#let exchange-rate = data.at("exchange_rate", default: none)
#let has-fx = exchange-rate != none
#let billing = data.at("billing", default: "hourly")
#let is-weekly = billing == "weekly"
#let payment-terms = data.at("payment_terms", default: "Net 30")
#let entries = data.entries

// Build rows with a running sub-total
#let rows = {
  let running = 0
  let result = ()
  for e in entries {
    let line-total = if is-weekly { e.amount } else { e.hours * e.rate }
    running = running + line-total
    result = result + ((entry: e, running: running, line-total: line-total),)
  }
  result
}

#let grand-total = if rows.len() > 0 { rows.last().running } else { 0 }

#let invoice-date = (
  datetime.today().display("[month repr:long] [day padding:none], [year]")
)

// Colours — from cfg.theme.colors with fallbacks
#let theme-colors = cfg.at("theme", default: (:)).at("colors", default: (:))
#let clr-body = rgb(theme-colors.at("body", default: "#464646"))
#let clr-label = rgb(theme-colors.at("label", default: "#323232"))
#let clr-panel-left = rgb(theme-colors.at("panel-left", default: "#f3f3f3"))
#let clr-panel-right = rgb(theme-colors.at("panel-right", default: "#6d98c2"))
#let clr-table-band = rgb(theme-colors.at("table-band", default: "#e8e8e8"))
#let clr-rule = rgb(theme-colors.at("rule", default: "#d2d2d2"))
#let clr-dim = rgb(theme-colors.at("dim", default: "#949494"))

// Page setup
#set page(paper: "a4", margin: (
  top: 2cm,
  bottom: 2cm,
  left: 2.2cm,
  right: 2.2cm,
))
#let theme-fonts = cfg.at("theme", default: (:)).at("fonts", default: (:))
#set text(
  ..(if "body" in theme-fonts { (font: theme-fonts.body) } else { (:) }),
  size: 9pt,
  fill: clr-body,
)
#set par(leading: 0.6em)

#let fmt-money(n, sym) = {
  let total-cents = calc.round(n * 100)
  let cents = calc.rem(total-cents, 100)
  let whole = int(total-cents / 100)
  let cents-str = if cents < 10 { "0" + str(cents) } else { str(cents) }
  let s = str(whole)
  let len = s.len()
  let formatted = ""
  for i in range(len) {
    if i > 0 and calc.rem(len - i, 3) == 0 { formatted += "," }
    formatted += s.at(i)
  }
  let amount = formatted + "." + cents-str
  if sym.len() > 1 { amount + " " + sym } else { sym + amount }
}

#let money(n) = fmt-money(n, currency)
#let money-to(n) = fmt-money(n, currency-to)

#let bank-sections-col(sections, col-align: left) = grid(
  columns: (auto, 1fr),
  column-gutter: 1.5em,
  row-gutter: 0.5em,
  align: col-align,
  ..sections
    .map(s => (
      [#text(weight: "bold", size: 8pt)[#s.label]],
      [#text(fill: clr-dim)[#s.value.split("\n").join(linebreak())]],
    ))
    .flatten(),
)

// Header
#grid(
  columns: (1.5fr, 3.5fr),
  rows: 6.5cm,
  align: (left + top, right + top),
  fill: (col, _) => if col == 0 { clr-panel-left } else { clr-panel-right },
  inset: (x: 1.4em, y: 1.4em),
  [
    #text(size: 8pt, weight: "bold", fill: clr-label)[Client]
    #v(0.4em)
    #text(weight: "bold")[#client-name]
    #if client-address != "" [
      #v(0.3em)
      #text(size: 9pt, fill: clr-dim)[#(
        client-address.split("\n").join(linebreak())
      )]
    ]
    #place(bottom + left)[
      #text(size: 8pt, weight: "bold", fill: clr-label)[Date]
      #v(0.4em)
      #invoice-date
    ]
  ],
  [
    #text(size: 13pt, weight: "bold", fill: white)[#upper(provider.name)]
    #linebreak()
    #text(size: 26pt, weight: "bold", fill: white)[INVOICE \##invoice-number]
    #let has-emblem = sys.inputs.at("has-emblem", default: "false") == "true"
    #if has-emblem {
      place(bottom + left, image("../emblem.svg", height: 2.5cm))
    }
    #place(bottom + right)[
      #text(size: 10pt, fill: white)[
        #provider.name \
        #provider.address \
        #provider.email
      ]
    ]
  ],
)

#v(1.5em)

// Line items
#let n = rows.len()
#let col-hdr(body) = text(size: 8pt, weight: "bold", fill: clr-label)[#body]
#table(
  columns: if has-fx { (1fr, auto, auto, auto, auto) } else if is-weekly {
    (1fr, auto, auto, auto)
  } else {
    (1fr, auto, auto, auto, auto)
  },
  align: if has-fx {
    (left, center, right, right, right)
  } else if is-weekly {
    (left, center, right, right)
  } else {
    (left, center, right, right, right)
  },
  stroke: (x, _) => if x > 0 { (left: 0.5pt + clr-rule) } else { none },
  inset: (x: 0.6em, y: 0.55em),
  fill: (_, row) => if row == 0 or row == n + 1 or calc.even(row) {
    clr-table-band
  } else {
    none
  },

  table.header(
    col-hdr[DESCRIPTION],
    col-hdr(if is-weekly { "WEEK" } else { "HOURS" }),
    ..if not is-weekly and not has-fx {
      (col-hdr("RATE (" + currency + "/hr)"),)
    } else {
      ()
    },
    col-hdr("PRICE" + (if has-fx { " (" + currency + ")" } else { "" })),
    ..if has-fx {
      (col-hdr("RATE (" + currency + " ⟶ " + currency-to + ")"),)
    } else {
      ()
    },
    col-hdr(if has-fx { "SUB TOTAL (" + currency-to + ")" } else {
      "SUB TOTAL"
    }),
  ),

  ..rows
    .map(r => {
      let qty = if is-weekly {
        [#r.entry.week]
      } else {
        [#r.entry.hours]
      }
      let rate-cell = if not is-weekly and not has-fx {
        ([#money(r.entry.rate)],)
      } else {
        ()
      }
      let base = (
        [#r.entry.description],
        qty,
        ..rate-cell,
        [#money(r.line-total)],
      )
      let fx = if has-fx {
        (
          [#str(exchange-rate)],
          [#money-to(r.running * exchange-rate)],
        )
      } else {
        ([#money(r.running)],)
      }
      base + fx
    })
    .flatten(),

  table.hline(stroke: 0.5pt + clr-rule),

  table.cell(colspan: if has-fx { 4 } else if is-weekly { 3 } else { 4 })[#text(
    weight: "bold",
  )[GRAND TOTAL]],
  [#text(weight: "bold")[#if has-fx {
    money-to(grand-total * exchange-rate)
  } else { money(grand-total) }]],
)

#v(1.5em)

#text(size: 9pt, fill: clr-dim)[Payment terms: #payment-terms]

#v(1em)

// Bank details
#grid(
  columns: (1fr,),
  gutter: 0pt,
  ..banks.map(bank => {
    let note = bank.at("note", default: none)
    block(
      width: 100%,
      stroke: 0.5pt + clr-rule,
      inset: (x: 1.2em, y: 1em),
    )[
      #underline(offset: 3pt, text(weight: "bold")[#bank.name])
      #if note != none [#h(0.4em)#text(size: 8pt, fill: clr-dim)[(#note)]]
      #v(0.8em)
      #let left-sections = bank.sections.filter(s => (
        s.at("position", default: "left") == "left"
      ))
      #let right-sections = bank.sections.filter(s => (
        s.at("position", default: "left") == "right"
      ))
      #grid(
        columns: (2fr, 1fr),
        column-gutter: 2em,
        bank-sections-col(left-sections),
        bank-sections-col(right-sections, col-align: right),
      )
    ]
  }),
)
