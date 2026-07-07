# Templates

**Category:** basics
**Difficulty:** Beginner

## Objective

Show Go's two template packages and the one difference that matters between them: `text/template` renders data into any text format (reports, config files, generated code), while `html/template` offers the *same API* plus contextual auto-escaping, so untrusted data can't break out of the HTML structure. Along the way: actions (`range`, `if`), custom functions via `FuncMap`, whitespace trimming, and composing documents from named blocks with `define`/`template`.

## Concepts Covered

- Template actions: `{{.Field}}`, `{{range}}`, `{{if}}/{{else}}`, `gt`/`len` built-ins
- `FuncMap` — registering custom functions (`upper`, a currency formatter) callable from template source
- `{{-` / `-}}` trim markers — keeping template-source formatting out of the rendered output
- `define` + `template` — named blocks and partials, the mechanism behind layout/page structures
- `html/template`'s contextual auto-escaping, demonstrated by pushing the same `<script>` payload through both packages
- Templates render to any `io.Writer` — stdout here, `http.ResponseWriter` or files in real programs

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
templates/
├── go.mod
├── main.go
├── Makefile
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

## Expected Output

```
--- text/template: report with range, if, and FuncMap ---
Invoice for ADA
- keyboard: 89.90 EUR (12 in stock)
- trackball: 54.50 EUR (SOLD OUT)
- desk mat: 19.00 EUR (3 in stock)

--- composition: define / template ---
== Order Summary ==
customer: Ada, items: 3

--- html/template: contextual auto-escaping ---
text/template: <p>comment from mallory: <script>alert("pwned")</script></p>
html/template: <p>comment from mallory: &lt;script&gt;alert(&#34;pwned&#34;)&lt;/script&gt;</p>
```

## Code Walkthrough

- `textReport` covers the daily-driver features in one template: `.Customer` field access, `{{range .Items}}` iterating the slice (inside the loop, `.` becomes the current item), `{{if gt .Stock 0}}` branching on data, and two `FuncMap` functions — note `Funcs` must be called *before* `Parse`, since parsing validates that every function referenced exists.
- The `{{-` and `-}}` markers trim adjacent whitespace and newlines. Without them, the line breaks that make the template source readable leak into the output as blank lines — compare the source's layout with the rendered invoice.
- `composedTemplates` builds a document from named blocks: `define "header"` declares a partial, and `{{template "header" "Order Summary"}}` invokes it with its own data (a plain string here — whatever you pass becomes the partial's `.`). `ExecuteTemplate` picks the entry point by name. This is the exact mechanism web apps use for layout + page templates, usually loading many files at once with `template.ParseFS` over an embedded filesystem (see [embed](../embed/)).
- `escapingContrast` is the security punchline. The template source is *identical* for both packages; only the import changes. `text/template` reproduces mallory's `<script>` verbatim — fine for a text report, an XSS hole in a web page. `html/template` parses the surrounding markup, understands the value lands in HTML text content, and escapes accordingly. The escaping is *contextual*: the same value placed inside an attribute, a URL, or a `<script>` block gets that context's rules, not one-size-fits-all entity encoding.

## Common Pitfalls

- **Using `text/template` to produce HTML.** The API is identical, so nothing warns you — until user input renders as markup. If the output is HTML, import `html/template`; the type system then also distinguishes pre-trusted fragments (`template.HTML`) from plain strings.
- **Wrapping untrusted input in `template.HTML` to "fix" escaping.** That type is an assertion that *you* already sanitized the value; applying it to user input reintroduces exactly the XSS the package prevents.
- **Calling `.Funcs` after `.Parse`.** Parse fails with `function "x" not defined` — registration must precede parsing because the parser checks the function table.
- **Ranging over a map when output must be stable.** Map iteration order is randomized (`text/template` sorts keys only for basic types); prefer slices when order matters, as this example does.
- **Ignoring `Execute`'s error.** Templates fail at render time too (nil field access, a FuncMap function returning an error) — and with an `http.ResponseWriter`, a partial page may already be written; render to a buffer first when you need all-or-nothing pages.

## References

- [text/template package docs](https://pkg.go.dev/text/template)
- [html/template package docs](https://pkg.go.dev/html/template)
- [html/template — security model](https://pkg.go.dev/html/template#hdr-Introduction)

## Next Steps

- [embed](../embed/) — `//go:embed` + `template.ParseFS`, how real projects ship template files
- [http-server](../http-server/) — where HTML templates typically get executed
- [io-readers-writers](../io-readers-writers/) — the `io.Writer` seam templates render into
