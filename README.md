[![PkgGoDev](https://pkg.go.dev/badge/github.com/localrivet/gopdf)](https://pkg.go.dev/github.com/localrivet/gopdf)

<!-- TODO: Add localrivet/gopdf specific badges (Build Status, Codecov, Report Card) if CI/setup is done -->

# gopdf (go-wkhtmltopdf Fork)

Golang commandline wrapper for `wkhtmltopdf`, extended with Markdown support and enhanced configuration.

**Note:** This package is a fork of the excellent [go-wkhtmltopdf](https://github.com/SebastiaanKlippert/go-wkhtmltopdf) library, originally created by Sebastiaan Klippert. Many thanks to Sebastiaan for the original work! This fork adds features specifically tailored for generating documents from Markdown with custom themes and layouts.

See http://wkhtmltopdf.org/index.html for the underlying `wkhtmltopdf` tool documentation.

| :warning: WARNING                                                                                                                  |
| :--------------------------------------------------------------------------------------------------------------------------------- |
| The underlying `wkhtmltopdf` tool is no longer maintained and now archived on GitHub. See https://wkhtmltopdf.org/status.html      |
| Consider alternatives like [Gotenberg](https://gotenberg.dev/) for new projects requiring robust, maintained PDF generation.       |
| This Go package fork (`gopdf`) may receive updates specific to LocalRivet's needs but relies on the archived `wkhtmltopdf` binary. |

# What and Why

This Go package provides a wrapper around the `wkhtmltopdf` command-line utility. It allows generating PDF documents from HTML content, making it suitable for creating invoices, reports, and other documents with customizable layouts using HTML/CSS.

Key features from the original library include:

- **Typed Options:** All `wkhtmltopdf` command-line options are represented as typed struct members, providing type safety and easier use with IDE code completion.
- **Input Flexibility:** Accepts multiple input sources, including URLs (`NewPage`) and `io.Reader` interfaces (`NewPageReader`) for processing in-memory HTML or local files. At most one input can be from an `io.Reader` (piped via stdin).
- **Concurrency:** Each `PDFGenerator` instance manages its own process and output buffer, suitable for server applications.
- **Output Options:** Generated PDFs can be retrieved from an internal buffer (`Bytes()`, `Buffer()`), written directly to a file (`WriteFile()`), or written to any `io.Writer` (`SetOutput()`).

## Fork Additions

This fork (`gopdf`) extends the original functionality with:

- **Markdown Input:** Directly generate PDFs from Markdown files using `NewMarkdownPage("path/to/file.md")`. The library handles the conversion from Markdown to HTML internally using `github.com/gomarkdown/markdown`.
- **Simplified Configuration:** Added convenience methods for common PDF elements:
  - `SetUserStyleSheet(path string)`: Apply a global CSS theme to all pages.
  - `SetCover(path string)`: Easily add a cover page from an HTML file.
  - `SetHeaderHTML(path string)` / `SetFooterHTML(path string)`: Set global header/footer HTML files.
  - `SetReplace(key, value string)`: Define global key-value pairs for substitution in headers/footers (e.g., `[author]`).
- **Cover Page Generation Helper:** Includes an example (`cmd/example/example.go`) demonstrating how to automatically generate a basic HTML cover page from the first H1/H2 titles in a Markdown file.
- **Content Skipping:** The `MarkdownPage` type includes a `SkipFirstH1H2 bool` flag. When set to `true`, the library attempts to skip the initial H1 and subsequent H2 block from the Markdown content when rendering the main document body (useful when that content is already used on a cover page).
- **Layout Control via CSS:** The Markdown-to-HTML conversion allows for CSS (applied via `SetUserStyleSheet`) to control page breaks (e.g., `page-break-before`, `page-break-after`, `page-break-inside`) for better document flow. Example rules are included in `testdata/theme.css`.

# Installation

```bash
go get -u github.com/localrivet/gopdf
```

Ensure the `wkhtmltopdf` binary (version 0.12.6 recommended) is installed and accessible in your system's PATH.

Alternatively, you can specify the path to the binary:

```go
wkhtmltopdf.SetPath("/path/to/your/wkhtmltopdf")
```

`gopdf` finds the path to `wkhtmltopdf` by:

- first looking in the current dir (Note: Go 1.19+ restricts this - see https://pkg.go.dev/os/exec@master#hdr-Executables_in_the_current_directory)
- looking in the PATH environment variable
- using the WKHTMLTOPDF_PATH environment variable

# Usage

## Basic Markdown to PDF Example

```go
package main

import (
	"log"

	wkhtmltopdf "github.com/localrivet/gopdf" // Use the new module path
)

func main() {
	// Initialize PDF generator
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		log.Fatalf("Failed to create PDF generator: %v", err)
	}

	// --- Configure Appearance ---
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeLetter)
	pdfg.MarginTopUnit.Set("25mm")    // ~1 inch
	pdfg.MarginBottomUnit.Set("25mm") // ~1 inch
	pdfg.MarginLeftUnit.Set("25mm")   // ~1 inch
	pdfg.MarginRightUnit.Set("25mm")  // ~1 inch

	// Apply a theme, footer, and header (optional)
	pdfg.SetUserStyleSheet("path/to/your/theme.css")
	pdfg.SetFooterHTML("path/to/your/footer.html")
	// pdfg.SetHeaderHTML("path/to/your/header.html") // Example

	// Add replacements for footer/header placeholders (e.g., [author])
	pdfg.SetReplace("author", "Your Name")

	// --- Add Content ---
	// Add a page directly from a Markdown file
	mdPage := wkhtmltopdf.NewMarkdownPage("path/to/your/document.md")
	// Optionally skip the first H1/H2 if used on a cover page
	// mdPage.SkipFirstH1H2 = true
	pdfg.AddPage(mdPage)

	// You can still add HTML pages or pages from readers
	// pdfg.AddPage(wkhtmltopdf.NewPage("https://example.com"))
	// pdfg.AddPage(wkhtmltopdf.NewPageReader(strings.NewReader("<h1>Hello</h1>")))

	// --- Generate ---
	err = pdfg.Create()
	if err != nil {
		log.Fatalf("Failed to create PDF: %v", err)
	}

	// --- Save ---
	err = pdfg.WriteFile("./output.pdf")
	if err != nil {
		log.Fatalf("Failed to write PDF file: %v", err)
	}

	log.Println("Successfully generated PDF: output.pdf")
}
```

## Example with Auto-Generated Cover Page

See `cmd/example/example.go` in this repository for a more detailed example that:

1.  Reads the input Markdown file.
2.  Extracts the first H1 and H2 titles.
3.  Generates a temporary HTML file for the cover page with specific styling.
4.  Uses `pdfg.SetCover()` to add the cover.
5.  Creates a `MarkdownPage` with `SkipFirstH1H2 = true` to avoid duplicating the title on the first content page.
6.  Generates the final PDF.

## Input from `io.Reader` (Stdin)

You can provide one document via an `io.Reader` using `NewPageReader`. This is useful for in-memory HTML or local files.

```go
html := "<html><body><h1>Hello from Reader</h1></body></html>"
pageReader := wkhtmltopdf.NewPageReader(strings.NewReader(html))
// Set page-specific options if needed
// pageReader.Zoom.Set(1.1)
pdfg.AddPage(pageReader)
```

# Saving to and loading from JSON

JSON serialization/deserialization allows preparing the PDF structure separately from generation.

- `Page` types save their input path/URL.
- `PageReader` types save their content as Base64.
- `MarkdownPage` types save their `InputPath` and `SkipFirstH1H2` flag. The content is **not** saved as Base64; the page is reconstructed from the `InputPath` upon deserialization using `NewPDFGeneratorFromJSON`.

Use `NewPDFPreparer` to create a `PDFGenerator` without needing `wkhtmltopdf` installed (e.g., client-side) and `NewPDFGeneratorFromJSON` to reconstruct it where `wkhtmltopdf` is available (e.g., server-side).

```go
// Client code
pdfg := wkhtmltopdf.NewPDFPreparer()
pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
pdfg.AddPage(wkhtmltopdf.NewMarkdownPage("report.md"))
// ... set other options ...

jb, err := pdfg.ToJSON()
// ... send jb to server ...

// Server code
pdfgFromServer, err := wkhtmltopdf.NewPDFGeneratorFromJSON(bytes.NewReader(jb))
if err != nil {
    log.Fatal(err)
}
err = pdfgFromServer.Create()
// ... handle PDF output ...
```

# Speed

The generation speed is primarily determined by `wkhtmltopdf` itself and the complexity/loading time of the source HTML/CSS/JS. The Go wrapper overhead is negligible.

---

_Original library by Sebastiaan Klippert._
_Fork enhancements by LocalRivet._
