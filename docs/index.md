# gopdf Documentation

Welcome to the documentation for `gopdf`, a Go library for generating PDFs from HTML and Markdown using the `wkhtmltopdf` command-line tool.

**Note:** This package is a fork of the excellent [go-wkhtmltopdf](https://github.com/SebastiaanKlippert/go-wkhtmltopdf) library, originally created by Sebastiaan Klippert. This fork adds features specifically tailored for generating documents from Markdown with custom themes and layouts.

| :warning: WARNING                                                                                                                  |
| :--------------------------------------------------------------------------------------------------------------------------------- |
| The underlying `wkhtmltopdf` tool is no longer maintained and now archived on GitHub. See https://wkhtmltopdf.org/status.html      |
| Consider alternatives like [Gotenberg](https://gotenberg.dev/) for new projects requiring robust, maintained PDF generation.       |
| This Go package fork (`gopdf`) may receive updates specific to LocalRivet's needs but relies on the archived `wkhtmltopdf` binary. |

## Key Features

- **HTML & Markdown Input:** Generate PDFs from URLs, local HTML files, HTML content via `io.Reader`, or directly from Markdown files.
- **Typed Options:** Provides Go structs and methods for nearly all `wkhtmltopdf` command-line options, ensuring type safety and ease of use.
- **Flexible Configuration:** Set global options for the entire document or page-specific options for individual inputs.
- **Simplified Theming & Layout:**
  - Apply global CSS stylesheets (`SetUserStyleSheet`).
  - Easily add cover pages (`SetCover`), headers (`SetHeaderHTML`), and footers (`SetFooterHTML`).
  - Use placeholder replacements in headers/footers (`SetReplace`).
- **Markdown Enhancements:**
  - Automatic Markdown-to-HTML conversion.
  - Option to skip the first H1/H2 block (`SkipFirstH1H2` flag) for use with generated cover pages.
  - Control page breaks via CSS in your theme file.
- **Concurrency Support:** Each `PDFGenerator` instance runs independently.
- **Flexible Output:** Get the PDF as bytes, write to a file, or stream to an `io.Writer`.
- **JSON Serialization:** Save and load generator configurations.

## Getting Started

1.  **Install `wkhtmltopdf`:** Download and install the `wkhtmltopdf` binary (version 0.12.6 recommended) for your system from the [official website](https://wkhtmltopdf.org/downloads.html) (or other sources, as it's archived). Ensure it's in your system's PATH.
2.  **Install `gopdf`:**
    ```bash
    go get -u github.com/localrivet/gopdf
    ```
3.  **Explore Usage:** See the [Usage Guide](usage.md) for examples.
4.  **API Reference:** Consult the [GoDoc](https://pkg.go.dev/github.com/localrivet/gopdf) or the [API Summary](api.md).

## Documentation Sections

- [Usage Guide](usage.md)
- [Markdown Features](markdown.md)
- [Configuration Options](configuration.md)
- [API Summary](api.md)
- [JSON Serialization](json.md) (Coming Soon)
