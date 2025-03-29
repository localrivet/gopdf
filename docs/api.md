# API Summary

This page provides a summary of the main public types and functions in the `gopdf` package. For complete details, please refer to the [GoDoc](https://pkg.go.dev/github.com/localrivet/gopdf).

## Core Type: `PDFGenerator`

The central struct for creating PDFs.

- `NewPDFGenerator() (*PDFGenerator, error)`: Creates a new generator and checks for the `wkhtmltopdf` executable.
- `NewPDFPreparer() *PDFGenerator`: Creates a new generator _without_ checking for the executable (useful for JSON serialization).
- `AddPage(p PageProvider)`: Adds an input page (HTML, Markdown, Reader) to the document. Applies global settings.
- `SetPages(p []PageProvider)`: Replaces all existing pages with the provided slice.
- `ResetPages()`: Removes all previously added pages.
- `Create() error`: Generates the PDF into the internal buffer.
- `CreateContext(ctx context.Context) error`: Generates the PDF, allowing for context cancellation.
- `Bytes() []byte`: Returns the generated PDF content from the internal buffer.
- `Buffer() *bytes.Buffer`: Returns a pointer to the internal output buffer.
- `WriteFile(filename string) error`: Writes the internal buffer content to the specified file.
- `SetOutput(w io.Writer)`: Sets an `io.Writer` for PDF output, bypassing the internal buffer.
- `SetStderr(w io.Writer)`: Sets an `io.Writer` to capture `wkhtmltopdf`'s stderr output.
- `ToJSON() ([]byte, error)`: Serializes the generator configuration (including page content for readers) to JSON.
- `NewPDFGeneratorFromJSON(jsonReader io.Reader) (*PDFGenerator, error)`: Creates a new generator from a JSON configuration.

**Global Configuration Methods on `PDFGenerator`:**

- `SetUserStyleSheet(path string)`
- `SetHeaderHTML(path string)`
- `SetFooterHTML(path string)`
- `SetReplace(key, value string)`
- `SetCover(path string)`
- Access global options directly (e.g., `pdfg.PageSize.Set(...)`, `pdfg.MarginTopUnit.Set(...)`). See `globalOptions` struct in GoDoc.
- Access cover options: `pdfg.Cover.Zoom.Set(...)`
- Access TOC options: `pdfg.TOC.Include = true`, `pdfg.TOC.DisableDottedLines.Set(...)`

## Page Input Types (`PageProvider` interface)

These types represent different sources for PDF content.

- **`Page`**: Represents an HTML page from a URL or local file path.
  - `NewPage(input string) *Page`: Constructor.
  - `Input`: The URL or file path.
  - `PageOptions`: Embedded struct for page-specific settings.
- **`PageReader`**: Represents an HTML page read from an `io.Reader`.
  - `NewPageReader(input io.Reader) *PageReader`: Constructor.
  - `Input`: The `io.Reader` providing HTML content.
  - `PageOptions`: Embedded struct for page-specific settings.
- **`MarkdownPage`**: Represents a page generated from a Markdown file.
  - `NewMarkdownPage(inputPath string) *MarkdownPage`: Constructor.
  - `InputPath`: The path to the Markdown file.
  - `SkipFirstH1H2 bool`: Flag to control skipping initial H1/H2 block.
  - `PageOptions`: Embedded struct for page-specific settings.

## Option Types

Most configuration options are set using helper types like:

- `stringOption`: For string values (e.g., `PageSize`, `Title`).
- `uintOption`: For unsigned integer values (e.g., `Dpi`, `MarginBottom`).
- `floatOption`: For float values (e.g., `Zoom`, `HeaderSpacing`).
- `boolOption`: For boolean flags (e.g., `Grayscale`, `NoCollate`).
- `mapOption`: For repeatable key-value options (e.g., `CustomHeader`, `Replace`).
- `sliceOption`: For repeatable value options (e.g., `Allow`, `RunScript`).

Each option type typically has a `Set(value)` method. Refer to GoDoc for specific option names within `globalOptions`, `pageOptions`, etc.

## Utility Functions

- `SetPath(path string)`: Globally sets the path to the `wkhtmltopdf` executable.
- `GetPath() string`: Retrieves the currently configured path to the executable.
