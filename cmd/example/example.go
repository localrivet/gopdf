package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	wkhtmltopdf "github.com/localrivet/gopdf" // Updated module path
)

// --- Helper function to create cover page ---
func createCoverPage(markdownPath, coverPath, author string) (string, error) { // Added author parameter
	file, err := os.Open(markdownPath)
	if err != nil {
		return "", fmt.Errorf("failed to open markdown file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var h1Title, h2Title string
	foundH1 := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !foundH1 && strings.HasPrefix(line, "# ") {
			h1Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			foundH1 = true
		} else if foundH1 && h2Title == "" && strings.HasPrefix(line, "## ") {
			h2Title = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			break // Found H1 and first H2, stop scanning
		} else if foundH1 && line != "" {
			// If we found H1 but the next non-empty line is not H2, stop looking for H2
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning markdown file: %w", err)
	}

	if h1Title == "" {
		return "", fmt.Errorf("no H1 title found in markdown file")
	}

	// Basic HTML structure for the cover page
	coverHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Cover</title>
    <style>
        html, body { height: 100%%; margin: 0; padding: 0; font-family: sans-serif; } /* Ensure full height */
        body { display: flex; flex-direction: column; justify-content: center; /* Vertical centering with flex */ min-height: 100vh; text-align: center; padding: 5%%; box-sizing: border-box; }
        .content { max-width: 80%%; margin-left: auto; margin-right: auto; } /* Horizontal centering with auto margins */
        h1 { font-size: 2.8em; margin-bottom: 0.5em; color: #111; line-height: 1.2; font-weight: bold; } /* Ensure H1 is bold */
        h2 { font-size: 1.6em; color: #555; font-weight: normal; margin-bottom: 1.5em; line-height: 1.3; }
        .author { font-size: 1.2em; color: #444; margin-top: 2em; }
    </style>
</head>
<body>
    <div class="content">
        <h1>%s</h1>
        <h2>%s</h2>
        <div class="author">%s</div> <!-- Added author -->
    </div>
</body>
</html>`, h1Title, h2Title, author) // Added author to Sprintf

	err = os.WriteFile(coverPath, []byte(coverHTML), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write cover page HTML: %w", err)
	}

	return coverPath, nil
}

// --- Main function ---
func main() {
	markdownPath := "testdata/testmd.md"
	coverPath := "cover_page.html" // Temporary cover file
	authorName := "LocalRivet"     // Define the author name

	// Create cover page HTML from Markdown titles
	_, err := createCoverPage(markdownPath, coverPath, authorName) // Pass author name
	if err != nil {
		log.Printf("Warning: Could not create cover page - %v. Proceeding without cover.", err)
		// Don't delete if creation failed
	} else {
		defer os.Remove(coverPath) // Schedule cleanup of temp cover file
	}

	// Initialize PDF generator
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		log.Fatalf("Failed to create PDF generator: %v", err)
	}

	// Set global options (optional)
	pdfg.Title.Set("Markdown Test Document")
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeLetter)
	pdfg.MarginTopUnit.Set("25mm")                // Explicitly set unit
	pdfg.MarginBottomUnit.Set("25mm")             // Explicitly set unit
	pdfg.MarginLeftUnit.Set("25mm")               // Explicitly set unit
	pdfg.MarginRightUnit.Set("25mm")              // Explicitly set unit
	pdfg.SetFooterHTML("testdata/footer.html")    // Add footer
	pdfg.SetUserStyleSheet("testdata/theme.css")  // Add theme CSS
	if _, err := os.Stat(coverPath); err == nil { // Check if cover file was created
		pdfg.SetCover(coverPath) // Set the cover page
	}
	pdfg.SetReplace("author", authorName) // Add replacement for footer author

	// Add Markdown page
	// Path is relative to the project root where 'go run' is executed
	mdPage := wkhtmltopdf.NewMarkdownPage(markdownPath) // Use variable
	mdPage.SkipFirstH1H2 = true                         // Set flag to skip H1/H2
	pdfg.AddPage(mdPage)

	// Create PDF
	err = pdfg.Create()
	if err != nil {
		log.Fatalf("Failed to create PDF: %v", err)
	}

	// Write PDF to file
	// Path is relative to the project root where 'go run' is executed
	outputFilename := "markdown_output.pdf"
	err = pdfg.WriteFile(outputFilename)
	if err != nil {
		log.Fatalf("Failed to write PDF file: %v", err)
	}

	log.Printf("Successfully generated PDF: %s\n", outputFilename)
}
