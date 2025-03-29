# Markdown Features

`gopdf` provides enhanced support for generating PDFs directly from Markdown files.

## `NewMarkdownPage`

The primary way to use Markdown input is via the `NewMarkdownPage` function:

```go
mdPage := wkhtmltopdf.NewMarkdownPage("path/to/your/document.md")
pdfg.AddPage(mdPage)
```

This creates a `MarkdownPage` object, which implements the `PageProvider` interface. When the PDF generator processes this page, it will:

1.  Read the content of the specified Markdown file (`InputPath`).
2.  Convert the Markdown content to HTML using the `github.com/gomarkdown/markdown` library (with common extensions enabled).
3.  Provide the resulting HTML to `wkhtmltopdf` via stdin.

This means you don't need to pre-convert your Markdown files to HTML yourself.

## Skipping Initial H1/H2 (`SkipFirstH1H2`)

Often, the main title (H1) and subtitle (H2) of a document are used to generate a separate cover page. To avoid duplicating this information on the first page of the main content, the `MarkdownPage` struct has a boolean flag:

```go
// Create the Markdown page provider
mdPage := wkhtmltopdf.NewMarkdownPage("path/to/your/document.md")

// Set the flag to true
mdPage.SkipFirstH1H2 = true

// Add the page to the generator
pdfg.AddPage(mdPage)
```

When `SkipFirstH1H2` is set to `true`, the `Reader()` method of `MarkdownPage` will attempt to:

1.  Scan the beginning of the Markdown file.
2.  Identify the first line starting with `# ` (H1).
3.  Identify the _next non-blank line_ starting with `## ` (H2).
4.  If both are found in sequence (allowing for blank lines between them), the content _before_ this H1/H2 block is discarded.
5.  If only an H1 is found before other non-blank content, only the H1 line(s) are skipped.
6.  The remaining Markdown content is then converted to HTML and passed to `wkhtmltopdf`.

This allows you to use the H1/H2 for a cover page (generated separately, see `cmd/example/example.go`) without having it repeated immediately on page 1 of the main document body.

**Note:** This skipping mechanism relies on simple prefix checking and might not cover all edge cases of complex Markdown structures around the initial headings.

## Styling Markdown Content

Since the Markdown is converted to standard HTML elements (`<h1>`, `<p>`, `<ul>`, `<strong>`, etc.), you can style the output using CSS via the `SetUserStyleSheet` method on the `PDFGenerator`.

```go
pdfg.SetUserStyleSheet("path/to/your/theme.css")
```

Your `theme.css` file can include rules for standard HTML tags to control fonts, margins, colors, and page breaks. See `testdata/theme.css` for an example.

### Controlling Page Breaks

You can use standard CSS page break properties in your theme file to influence layout:

- `h1 { page-break-before: always; }` (Start each H1 on a new page)
- `h1, h2, h3, h4, h5, h6 { page-break-after: avoid; }` (Try not to break after a heading)
- `p, li, blockquote { page-break-inside: avoid; }` (Try to keep paragraphs, list items, etc., together on one page)
- `hr { display: none; }` (Completely hide horizontal rules if desired)

Experiment with these properties in your CSS to achieve the desired document flow.
