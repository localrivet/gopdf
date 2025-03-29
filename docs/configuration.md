# Configuration Options

`gopdf` provides several ways to configure the PDF generation process, mirroring the command-line options of `wkhtmltopdf`.

## Global Options

These options apply to the entire PDF document and are set directly on the `PDFGenerator` object.

**Common Global Options:**

- `pdfg.PageSize.Set(wkhtmltopdf.PageSizeLetter)`: Sets the paper size (e.g., `PageSizeA4`, `PageSizeLetter`). Constants are defined in the package.
- `pdfg.Orientation.Set(wkhtmltopdf.OrientationLandscape)`: Sets page orientation (`OrientationPortrait` or `OrientationLandscape`).
- `pdfg.MarginTopUnit.Set("20mm")`: Sets the top margin (similarly `MarginBottomUnit`, `MarginLeftUnit`, `MarginRightUnit`). Accepts units like `mm`, `cm`, `in`. If using the non-unit versions (`MarginTop.Set(20)`), the default unit is `mm`.
- `pdfg.Dpi.Set(300)`: Sets the DPI (dots per inch).
- `pdfg.Grayscale.Set(true)`: Generates the PDF in grayscale.
- `pdfg.Title.Set("My Document Title")`: Sets the document title metadata.

**Convenience Setters (Global Defaults):**

These methods provide an easy way to set common options that will be applied to all pages added _after_ the setter is called, unless overridden by page-specific options.

- `pdfg.SetUserStyleSheet(path string)`: Specifies a global CSS file to apply to all HTML inputs (including converted Markdown). Corresponds to `--user-style-sheet`.
- `pdfg.SetHeaderHTML(path string)`: Sets a default HTML file to use for page headers. Corresponds to `--header-html`.
- `pdfg.SetFooterHTML(path string)`: Sets a default HTML file to use for page footers. Corresponds to `--footer-html`.
- `pdfg.SetReplace(key, value string)`: Defines a key-value pair for placeholder substitution within headers and footers (e.g., set `[author]` placeholder). Corresponds to `--replace`. Multiple calls add multiple replacements.
- `pdfg.SetCover(path string)`: Specifies an HTML file to use as a cover page. Corresponds to the `cover` command.

## Page Options

These options apply only to a specific input page (`Page`, `PageReader`, or `MarkdownPage`). They are accessed via the `PageOptions` field embedded within the page struct.

```go
// Example for a Page object
page := wkhtmltopdf.NewPage("https://example.com")
page.Zoom.Set(1.5) // Set zoom level for this page only
page.FooterCenter.Set("Page [page]") // Set a specific footer for this page
pdfg.AddPage(page)

// Example for a MarkdownPage object
mdPage := wkhtmltopdf.NewMarkdownPage("report.md")
mdPage.UserStyleSheet.Set("specific-style.css") // Override global stylesheet for this page
mdPage.SkipFirstH1H2 = true // Specific flag for MarkdownPage
pdfg.AddPage(mdPage)
```

**Common Page Options:**

- `page.Zoom.Set(1.1)`: Sets the zoom factor for this page.
- `page.UserStyleSheet.Set("path/to/page.css")`: Sets a specific stylesheet for this page, overriding the global one set by `SetUserStyleSheet`.
- `page.EnableJavascript.Set(true)` / `page.DisableJavascript.Set(true)`: Control JavaScript execution for this page.
- `page.HeaderHTML.Set(...)` / `page.FooterHTML.Set(...)`: Set page-specific header/footer HTML, overriding global settings.
- `page.HeaderSpacing.Set(10)` / `page.FooterSpacing.Set(5)`: Set spacing (in mm) between header/footer and content.
- `page.Replace.Set("section", "Introduction")`: Set page-specific replacements for header/footer placeholders.

## Cover and TOC Options

Cover pages and Table of Contents (TOC) also have their own specific options that can be accessed via the `PDFGenerator`:

- `pdfg.Cover.Input = "path/to/cover.html"` (or use `SetCover`)
- `pdfg.Cover.Zoom.Set(0.8)` (Cover pages have their own `pageOptions`)
- `pdfg.TOC.Include = true` (Enable the TOC)
- `pdfg.TOC.DisableDottedLines.Set(true)`
- `pdfg.TOC.TocHeaderText.Set("Table of Contents")`
- `pdfg.TOC.HeaderHTML.Set("path/to/toc_header.html")` (TOC can have its own header/footer)

## Finding All Options

For a complete list of all available global, page, cover, and TOC options, refer to the GoDoc documentation for the following structs:

- `globalOptions`
- `outlineOptions`
- `pageOptions`
- `headerAndFooterOptions`
- `tocOptions`

These structs directly map to the command-line flags available in `wkhtmltopdf`.
