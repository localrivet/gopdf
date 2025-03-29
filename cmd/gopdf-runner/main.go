package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	wk "github.com/localrivet/gopdf" // Use our forked module path
)

// Simple map flag for replacements
type replaceMap map[string]string

func (r *replaceMap) String() string {
	// Just return a placeholder, actual value isn't important for flag package
	return "key=value"
}

func (r *replaceMap) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format for replace flag: %s. Use key=value", value)
	}
	(*r)[parts[0]] = parts[1]
	return nil
}

func main() {
	// --- Define command-line flags ---
	input := flag.String("input", "", "The raw Markdown or HTML content string (required)") // Renamed back, accepts content
	outputPath := flag.String("output", "", "Path for the generated PDF file (required)")
	inputType := flag.String("inputType", "markdown", "Type of input content ('markdown' or 'html')")
	themePath := flag.String("theme", "", "Path to CSS theme file (optional)")
	footerPath := flag.String("footer", "", "Path to footer HTML file (optional)")
	headerPath := flag.String("header", "", "Path to header HTML file (optional)")
	coverPath := flag.String("cover", "", "Path to cover HTML file (optional)")
	skipH1H2 := flag.Bool("skipH1H2", false, "Skip first H1/H2 block in Markdown input (for cover pages)")
	marginTop := flag.String("marginTop", "", "Top margin (e.g., '25mm', '1in') (optional)")
	marginBottom := flag.String("marginBottom", "", "Bottom margin (e.g., '25mm', '1in') (optional)")
	marginLeft := flag.String("marginLeft", "", "Left margin (e.g., '25mm', '1in') (optional)")
	marginRight := flag.String("marginRight", "", "Right margin (e.g., '25mm', '1in') (optional)")
	pageSize := flag.String("pageSize", "", "Page size (e.g., 'Letter', 'A4') (optional)")
	orientation := flag.String("orientation", "", "Page orientation ('Portrait' or 'Landscape') (optional)")
	title := flag.String("title", "", "Document title metadata (optional)")

	replacements := make(replaceMap)
	flag.Var(&replacements, "replace", "Key-value pair for header/footer replacement (key=value). Can be specified multiple times.")

	flag.Parse()

	// --- Validate required flags ---
	if *input == "" { // Use input
		log.Fatal("Error: -input flag is required") // Use correct flag name in message
	}
	if *outputPath == "" {
		log.Fatal("Error: -output flag is required")
	}

	// --- Initialize PDF generator ---
	pdfg, err := wk.NewPDFGenerator()
	if err != nil {
		log.Fatalf("Error creating PDF generator: %v", err)
	}

	// --- Apply options from flags ---
	if *title != "" {
		pdfg.Title.Set(*title)
	}
	if *pageSize != "" {
		pdfg.PageSize.Set(*pageSize)
	}
	if *orientation != "" {
		pdfg.Orientation.Set(*orientation)
	}
	if *marginTop != "" {
		pdfg.MarginTopUnit.Set(*marginTop)
	}
	if *marginBottom != "" {
		pdfg.MarginBottomUnit.Set(*marginBottom)
	}
	if *marginLeft != "" {
		pdfg.MarginLeftUnit.Set(*marginLeft)
	}
	if *marginRight != "" {
		pdfg.MarginRightUnit.Set(*marginRight)
	}
	if *themePath != "" {
		pdfg.SetUserStyleSheet(*themePath)
	}
	if *footerPath != "" {
		pdfg.SetFooterHTML(*footerPath)
	}
	if *headerPath != "" {
		pdfg.SetHeaderHTML(*headerPath)
	}
	if *coverPath != "" {
		// Check if cover file exists before setting, prevent wkhtmltopdf error
		if _, err := os.Stat(*coverPath); err == nil {
			pdfg.SetCover(*coverPath)
		} else {
			log.Printf("Warning: Cover file not found at %s, skipping cover.", *coverPath)
		}
	}
	for k, v := range replacements {
		pdfg.SetReplace(k, v)
	}

	// --- Add input page ---
	var pageProvider wk.PageProvider
	var tempFile *os.File // For temporary markdown file

	switch strings.ToLower(*inputType) {
	case "markdown":
		// Create a temporary file for markdown content
		tmpFile, err := os.CreateTemp("", "input-*.md")
		if err != nil {
			log.Fatalf("Error creating temporary markdown file: %v", err)
		}
		tempFile = tmpFile // Store to remove later
		if _, err := tmpFile.WriteString(*input); err != nil {
			tmpFile.Close()           // Close on error
			os.Remove(tmpFile.Name()) // Attempt cleanup
			log.Fatalf("Error writing to temporary markdown file: %v", err)
		}
		if err := tmpFile.Close(); err != nil {
			os.Remove(tmpFile.Name()) // Attempt cleanup
			log.Fatalf("Error closing temporary markdown file: %v", err)
		}

		// Use the temporary file path with NewMarkdownPage
		mdPage := wk.NewMarkdownPage(tmpFile.Name())
		mdPage.SkipFirstH1H2 = *skipH1H2
		pageProvider = mdPage

	case "html":
		// Use NewPageReader for HTML content string
		pageProvider = wk.NewPageReader(strings.NewReader(*input))
	default:
		log.Fatalf("Error: Invalid -inputType '%s'. Use 'markdown' or 'html'.", *inputType)
	}

	// Defer removal of temporary file if it was created
	if tempFile != nil {
		defer os.Remove(tempFile.Name())
	}

	pdfg.AddPage(pageProvider)

	// --- Generate PDF ---
	err = pdfg.Create()
	if err != nil {
		log.Fatalf("Error creating PDF: %v", err)
	}

	// --- Save PDF ---
	err = pdfg.WriteFile(*outputPath)
	if err != nil {
		log.Fatalf("Error writing PDF file: %v", err)
	}

	// --- Output success message (stdout) ---
	// MCP server will read this to know the output path
	fmt.Println(*outputPath)
}
