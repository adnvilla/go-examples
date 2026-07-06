// Demonstrates Go's two template packages and the one difference that
// matters: text/template renders data into any text format (reports, config,
// code), while html/template has the same API plus contextual auto-escaping —
// untrusted data cannot break out of the HTML structure. Also covers actions
// (range, if), custom functions via FuncMap, and composing templates with
// define/template.
package main

import (
	"fmt"
	htmltemplate "html/template"
	"os"
	"strings"
	texttemplate "text/template"
)

type Item struct {
	Name  string
	Price float64
	Stock int
}

type Order struct {
	Customer string
	Items    []Item
}

var order = Order{
	Customer: "Ada",
	Items: []Item{
		{Name: "keyboard", Price: 89.90, Stock: 12},
		{Name: "trackball", Price: 54.50, Stock: 0},
		{Name: "desk mat", Price: 19.00, Stock: 3},
	},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := textReport(); err != nil {
		return err
	}
	if err := composedTemplates(); err != nil {
		return err
	}
	return escapingContrast()
}

// textReport renders a plain-text invoice: field access, range with $-scoped
// outer variables, if/else on data, and two custom functions registered
// through FuncMap. The `-` trim markers eat the newlines that template
// source formatting would otherwise leak into the output.
func textReport() error {
	fmt.Println("--- text/template: report with range, if, and FuncMap ---")

	funcs := texttemplate.FuncMap{
		"upper": strings.ToUpper,
		"eur":   func(v float64) string { return fmt.Sprintf("%.2f EUR", v) },
	}

	const src = `Invoice for {{upper .Customer}}
{{- range .Items}}
- {{.Name}}: {{eur .Price}} {{if gt .Stock 0}}({{.Stock}} in stock){{else}}(SOLD OUT){{end}}
{{- end}}
`
	tmpl, err := texttemplate.New("invoice").Funcs(funcs).Parse(src)
	if err != nil {
		return fmt.Errorf("parsing invoice template: %w", err)
	}
	if err := tmpl.Execute(os.Stdout, order); err != nil {
		return fmt.Errorf("rendering invoice: %w", err)
	}
	return nil
}

// composedTemplates splits a document into named blocks: a layout that
// {{template}}s a header partial. Real projects parse many files this way
// (template.ParseFS pairs naturally with //go:embed).
func composedTemplates() error {
	fmt.Println("\n--- composition: define / template ---")

	const src = `
{{- define "header" -}}
== {{.}} ==
{{- end -}}

{{- define "page" -}}
{{template "header" "Order Summary"}}
customer: {{.Customer}}, items: {{len .Items}}
{{- end -}}`

	tmpl, err := texttemplate.New("doc").Parse(src)
	if err != nil {
		return fmt.Errorf("parsing composed templates: %w", err)
	}
	if err := tmpl.ExecuteTemplate(os.Stdout, "page", order); err != nil {
		return fmt.Errorf("rendering page: %w", err)
	}
	fmt.Println()
	return nil
}

// escapingContrast pushes the same hostile input through both packages.
// text/template reproduces it verbatim; html/template escapes it according
// to where it lands in the markup, neutralizing the script injection.
func escapingContrast() error {
	fmt.Println("\n--- html/template: contextual auto-escaping ---")

	const src = `<p>comment from {{.User}}: {{.Comment}}</p>`
	hostile := struct {
		User    string
		Comment string
	}{
		User:    "mallory",
		Comment: `<script>alert("pwned")</script>`,
	}

	textTmpl, err := texttemplate.New("raw").Parse(src)
	if err != nil {
		return fmt.Errorf("parsing text version: %w", err)
	}
	fmt.Print("text/template: ")
	if err := textTmpl.Execute(os.Stdout, hostile); err != nil {
		return fmt.Errorf("rendering text version: %w", err)
	}
	fmt.Println()

	htmlTmpl, err := htmltemplate.New("safe").Parse(src)
	if err != nil {
		return fmt.Errorf("parsing html version: %w", err)
	}
	fmt.Print("html/template: ")
	if err := htmlTmpl.Execute(os.Stdout, hostile); err != nil {
		return fmt.Errorf("rendering html version: %w", err)
	}
	fmt.Println()
	return nil
}
