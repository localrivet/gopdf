# Usage Guide

This guide provides basic examples of how to use the `gopdf` library.

## Prerequisites

- `wkhtmltopdf` binary installed and in PATH.
- `gopdf` library installed (`go get -u github.com/localrivet/gopdf`).

## Example 1: PDF from URL

```go
package main

import (
	"log"

	wk "github.com/localrivet/gopdf"
)

func main() {
	pdfg, err := wk.NewPDFGenerator()
	if err != nil {
		log.Fatal(err)
	}

	// Add a page from a URL
	pdfg.AddPage(wk.NewPage("https://example.com"))

	// Set some global options
	pdfg.PageSize.Set(wk.PageSizeA4)
	pdfg.Orientation.Set(wk.OrientationPortrait)

	err = pdfg.Create()
	if err != nil {
		log.Fatal(err)
	}

	err = pdfg.WriteFile("./url_example.pdf")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Generated url_example.pdf")
}
```

## Example 2: PDF from HTML String

```go
package main

import (
	"log"
	"strings"

	wk "github.com/localrivet/gopdf"
)

func main() {
	pdfg, err := wk.NewPDFGenerator()
	if err != nil {
		log.Fatal(err)
	}

	// HTML content
	htmlContent := `
		<!DOCTYPE html>
		<html>
		<head><title>Simple HTML</title></head>
		<body><h1>Hello PDF!</h1><p>This is generated from an HTML string.</p></body>
		</html>
	`

	// Add page from reader
	pdfg.AddPage(wk.NewPageReader(strings.NewReader(htmlContent)))

	err = pdfg.Create()
	if err != nil {
		log.Fatal(err)
	}

	err = pdfg.WriteFile("./html_string_example.pdf")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Generated html_string_example.pdf")
}
```

## Example 3: PDF from Markdown File (Basic)

This example uses the new `MarkdownPage` type.

```go
package main

import (
	"log"

	wk "github.com/localrivet/gopdf"
)

func main() {
	pdfg, err := wk.NewPDFGenerator()
	if err != nil {
		log.Fatal(err)
	}

	// Add page from a Markdown file
	// Assumes 'mydocument.md' exists in the same directory
	mdPage := wk.NewMarkdownPage("mydocument.md")
	pdfg.AddPage(mdPage)

	// Set global options
	pdfg.MarginTopUnit.Set("20mm")
	pdfg.MarginBottomUnit.Set("20mm")

	err = pdfg.Create()
	if err != nil {
		log.Fatal(err)
	}

	err = pdfg.WriteFile("./markdown_example.pdf")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Generated markdown_example.pdf")
}
```

## Example 4: Markdown with Theme, Footer, and Cover

This demonstrates using the convenience setters. See `cmd/example/example.go` in the repository for a version that also auto-generates the cover page HTML.

```go
package main

import (
	"log"
	"os" // Needed for os.Stat

	wk "github.com/localrivet/gopdf"
)

func main() {
	pdfg, err := wk.NewPDFGenerator()
	if err != nil {
		log.Fatal(err)
	}

	// --- Configuration ---
	authorName := "Your Name Here"
	markdownInputPath := "path/to/your/document.md"
	coverHTMLPath := "path/to/your/cover.html" // Pre-generated cover HTML
	footerHTMLPath := "path/to/your/footer.html"
	themeCSSPath := "path/to/your/theme.css"
	outputPDFPath := "./themed_markdown.pdf"

	// Set global options
	pdfg.PageSize.Set(wk.PageSizeLetter)
	pdfg.MarginTopUnit.Set("25mm")
	pdfg.MarginBottomUnit.Set("25mm")
	pdfg.MarginLeftUnit.Set("25mm")
	pdfg.MarginRightUnit.Set("25mm")

	// Apply theme, footer, header (optional), cover, replacements
	pdfg.SetUserStyleSheet(themeCSSPath)
	pdfg.SetFooterHTML(footerHTMLPath)
	pdfg.SetReplace("author", authorName) // For footer placeholder [author]

	// Set cover page if the HTML file exists
	if _, err := os.Stat(coverHTMLPath); err == nil {
		pdfg.SetCover(coverHTMLPath)
	} else {
		log.Printf("Warning: Cover page HTML not found at %s", coverHTMLPath)
	}

	// --- Add Content ---
	mdPage := wk.NewMarkdownPage(markdownInputPath)
	// If cover page was set, skip H1/H2 in main content
	if _, err := os.Stat(coverHTMLPath); err == nil {
		mdPage.SkipFirstH1H2 = true
	}
	pdfg.AddPage(mdPage)

	// --- Generate & Save ---
	err = pdfg.Create()
	if err != nil {
		log.Fatalf("Failed to create PDF: %v", err)
	}

	err = pdfg.WriteFile(outputPDFPath)
	if err != nil {
		log.Fatalf("Failed to write PDF file: %v", err)
	}

	log.Printf("Successfully generated PDF: %s\n", outputPDFPath)
}
```

See other documentation sections for more details on specific features like Markdown handling and configuration options.
