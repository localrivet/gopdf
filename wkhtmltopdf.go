// Package wkhtmltopdf provides Go bindings for the wkhtmltopdf command-line tool,
// allowing generation of PDFs from HTML content.
//
// This package is a fork of github.com/SebastiaanKlippert/go-wkhtmltopdf,
// originally created by Sebastiaan Klippert, with added features for Markdown
// processing and enhanced configuration options by LocalRivet.
package wkhtmltopdf

import (
	"bufio" // Added for scanner in Reader
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// the cached mutexed path as used by findPath()
type stringStore struct {
	val string
	sync.Mutex
}

func (ss *stringStore) Get() string {
	ss.Lock()
	defer ss.Unlock()
	return ss.val
}

func (ss *stringStore) Set(s string) {
	ss.Lock()
	ss.val = s
	ss.Unlock()
}

var binPath stringStore

// SetPath sets the path to wkhtmltopdf
func SetPath(path string) {
	binPath.Set(path)
}

// GetPath gets the path to wkhtmltopdf
func GetPath() string {
	return binPath.Get()
}

// Page is the input struct for each page
type Page struct {
	Input string
	PageOptions
}

// Options returns the PageOptions associated with this Page.
func (p *Page) Options() *PageOptions {
	return &p.PageOptions
}

// InputFile returns the input string and is part of the page interface
func (p *Page) InputFile() string {
	return p.Input
}

// Args returns the argument slice and is part of the page interface
func (p *Page) Args() []string {
	return p.PageOptions.Args()
}

// Reader returns the io.Reader and is part of the page interface
func (p *Page) Reader() io.Reader {
	return nil
}

// NewPage creates a new input page from a local or web resource (filepath or URL)
func NewPage(input string) *Page {
	return &Page{
		Input:       input,
		PageOptions: NewPageOptions(),
	}
}

// PageReader is one input page (a HTML document) that is read from an io.Reader
// You can add only one Page from a reader
type PageReader struct {
	Input io.Reader
	PageOptions
}

// Options returns the PageOptions associated with this PageReader.
func (pr *PageReader) Options() *PageOptions {
	return &pr.PageOptions
}

// InputFile returns the input string and is part of the page interface
func (pr *PageReader) InputFile() string {
	return "-"
}

// Args returns the argument slice and is part of the page interface
func (pr *PageReader) Args() []string {
	return pr.PageOptions.Args()
}

// Reader returns the io.Reader and is part of the page interface
func (pr *PageReader) Reader() io.Reader {
	return pr.Input
}

// NewPageReader creates a new PageReader from an io.Reader
func NewPageReader(input io.Reader) *PageReader {
	return &PageReader{
		Input:       input,
		PageOptions: NewPageOptions(),
	}
}

// MarkdownPage represents a page created from a Markdown file.
// The Markdown content will be converted to HTML internally before being passed to wkhtmltopdf.
// It implements the PageProvider interface.
type MarkdownPage struct {
	// InputPath is the filesystem path to the Markdown file.
	InputPath string
	// SkipFirstH1H2, if true, attempts to remove the first H1 heading and the
	// immediately following H2 heading (if present) from the Markdown content
	// before converting to HTML. This is useful if the H1/H2 are used for a
	// separate cover page.
	SkipFirstH1H2 bool
	PageOptions
	htmlCache []byte // Cache for the converted HTML
	readErr   error  // Store error during file read/conversion
}

// Options returns the PageOptions associated with this MarkdownPage.
func (mp *MarkdownPage) Options() *PageOptions {
	return &mp.PageOptions
}

// NewMarkdownPage creates a new MarkdownPage provider from a Markdown file path.
// By default, SkipFirstH1H2 is false.
func NewMarkdownPage(inputPath string) *MarkdownPage {
	return &MarkdownPage{
		InputPath:     inputPath,
		SkipFirstH1H2: false, // Default to false
		PageOptions:   NewPageOptions(),
	}
}

// Args returns the argument slice and is part of the page interface
func (mp *MarkdownPage) Args() []string {
	return mp.PageOptions.Args()
}

// InputFile returns "-" as Markdown is converted and piped via stdin.
func (mp *MarkdownPage) InputFile() string {
	return "-"
}

// Reader reads the Markdown file, converts it to HTML, and returns it as an io.Reader.
// It caches the result to avoid re-reading and re-converting.
// If SkipFirstH1H2 is true, it attempts to skip the first H1 and subsequent H2 block.
func (mp *MarkdownPage) Reader() io.Reader {
	if mp.htmlCache != nil || mp.readErr != nil {
		if mp.readErr != nil {
			// Return a reader that immediately returns the stored error
			return &errorReader{err: mp.readErr}
		}
		return bytes.NewReader(mp.htmlCache)
	}

	mdBytesAll, err := os.ReadFile(mp.InputPath)
	if err != nil {
		mp.readErr = fmt.Errorf("failed to read markdown file %s: %w", mp.InputPath, err)
		return &errorReader{err: mp.readErr}
	}

	mdBytesToParse := mdBytesAll // Default to parsing all bytes
	if mp.SkipFirstH1H2 {
		// Find the end of the first H1/H2 block to skip it
		scanner := bufio.NewScanner(bytes.NewReader(mdBytesAll))
		var byteOffset int
		foundH1 := false
		skipped := false
		linesToSkip := 0 // Count lines belonging to H1/H2 block

		for scanner.Scan() {
			line := scanner.Text() // Keep original line endings
			trimmedLine := strings.TrimSpace(line)
			// Use scanner.Bytes() for accurate length with potentially different line endings
			lineLen := len(scanner.Bytes()) + 1 // +1 for newline character

			if !foundH1 && strings.HasPrefix(trimmedLine, "# ") {
				foundH1 = true
				byteOffset += lineLen
				linesToSkip++
			} else if foundH1 && strings.HasPrefix(trimmedLine, "## ") {
				// Found H2 immediately after H1 (or whitespace)
				byteOffset += lineLen
				linesToSkip++
				mdBytesToParse = mdBytesAll[byteOffset:]
				skipped = true
				break
			} else if foundH1 && trimmedLine != "" {
				// Found H1, but the next non-empty line wasn't H2
				mdBytesToParse = mdBytesAll[byteOffset:] // Skip only the H1 line(s)
				skipped = true
				break
			} else if foundH1 && trimmedLine == "" { // Allow whitespace between H1 and H2
				byteOffset += lineLen
				linesToSkip++ // Count blank lines as part of the block to skip
			} else if !foundH1 { // Before H1
				byteOffset += lineLen // Accumulate offset but don't count as skipped lines yet
			} else {
				// Should not happen if logic is correct, but break just in case
				break
			}
		}
		if !skipped {
			// If we didn't find H1 or H2 as expected, parse everything
			// (or log a warning, but for now just parse all)
			mdBytesToParse = mdBytesAll
		} else if err := scanner.Err(); err != nil {
			// Handle potential scanner error after finding skip point
			mp.readErr = fmt.Errorf("error scanning markdown to skip H1/H2: %w", err)
			return &errorReader{err: mp.readErr}
		}
	}

	// Configure markdown parser and renderer
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(mdBytesToParse) // Parse the potentially truncated bytes

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	// Render the main markdown body
	bodyContent := markdown.Render(doc, renderer)

	// Wrap in basic HTML structure WITHOUT injecting styles here.
	// Styling will be handled by the external CSS file set via SetUserStyleSheet.
	var fullHTML bytes.Buffer
	fullHTML.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title></title></head><body>") // Removed <style> block
	fullHTML.Write(bodyContent)
	fullHTML.WriteString("</body></html>")

	mp.htmlCache = fullHTML.Bytes()
	return bytes.NewReader(mp.htmlCache)
}

// Helper type to return an error from an io.Reader
type errorReader struct {
	err error
}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, er.err
}

// PageProvider is the interface which provides a single input page.
// Implemented by Page, PageReader, and MarkdownPage.
type PageProvider interface {
	Args() []string
	InputFile() string
	Reader() io.Reader
	Options() *PageOptions // Added method to access PageOptions
}

// PageOptions are options for each input page
type PageOptions struct {
	pageOptions
	headerAndFooterOptions
}

// Args returns the argument slice
func (po *PageOptions) Args() []string {
	return append(append([]string{}, po.pageOptions.Args()...), po.headerAndFooterOptions.Args()...)
}

// NewPageOptions returns a new PageOptions struct with all options
func NewPageOptions() PageOptions {
	return PageOptions{
		pageOptions:            newPageOptions(),
		headerAndFooterOptions: newHeaderAndFooterOptions(),
	}
}

// cover page
type cover struct {
	Input string
	pageOptions
}

// table of contents
type toc struct {
	Include bool
	allTocOptions
}

type allTocOptions struct {
	pageOptions
	tocOptions
	headerAndFooterOptions
}

// PDFGenerator is the main wkhtmltopdf struct, always use NewPDFGenerator to obtain a new PDFGenerator struct
type PDFGenerator struct {
	globalOptions
	outlineOptions

	Cover      cover
	TOC        toc
	OutputFile string //filename to write to, default empty (writes to internal buffer)

	// Global settings applied to pages added after these are set
	userStyleSheetPath string
	headerHTMLPath     string
	footerHTMLPath     string
	replace            mapOption // Added global replace map

	binPath   string
	outbuf    bytes.Buffer
	outWriter io.Writer
	stdErr    io.Writer
	pages     []PageProvider // Keep track of added pages
}

// Args returns the commandline arguments as a string slice
func (pdfg *PDFGenerator) Args() []string {
	args := append([]string{}, pdfg.globalOptions.Args()...)
	args = append(args, pdfg.outlineOptions.Args()...)
	if pdfg.Cover.Input != "" {
		args = append(args, "cover")
		args = append(args, pdfg.Cover.Input)
		args = append(args, pdfg.Cover.pageOptions.Args()...)
	}
	if pdfg.TOC.Include {
		args = append(args, "toc")
		args = append(args, pdfg.TOC.pageOptions.Args()...)
		args = append(args, pdfg.TOC.tocOptions.Args()...)
		args = append(args, pdfg.TOC.headerAndFooterOptions.Args()...)
	}
	for _, page := range pdfg.pages {
		args = append(args, "page")
		args = append(args, page.InputFile())
		args = append(args, page.Args()...)
	}
	if pdfg.OutputFile != "" {
		args = append(args, pdfg.OutputFile)
	} else {
		args = append(args, "-")
	}
	return args
}

// ArgString returns Args as a single string
func (pdfg *PDFGenerator) ArgString() string {
	return strings.Join(pdfg.Args(), " ")
}

// AddPage adds a new input page to the document.
// A page is an input HTML page, it can span multiple pages in the output document.
// It is a Page when read from file or URL, a PageReader when read from memory,
// or a MarkdownPage when read from a Markdown file.
//
// It applies the generator's global settings (stylesheet, header, footer, replacements)
// to the page's options if they are not already set on the page itself.
// Page-specific options always take precedence over global settings.
func (pdfg *PDFGenerator) AddPage(p PageProvider) {
	opts := p.Options()

	// Apply global stylesheet if not set on page
	if pdfg.userStyleSheetPath != "" && opts.UserStyleSheet.value == "" {
		opts.UserStyleSheet.Set(pdfg.userStyleSheetPath)
	}

	// Apply global header if not set on page
	if pdfg.headerHTMLPath != "" && opts.HeaderHTML.value == "" {
		opts.HeaderHTML.Set(pdfg.headerHTMLPath)
	}

	// Apply global footer if not set on page
	if pdfg.footerHTMLPath != "" && opts.FooterHTML.value == "" {
		opts.FooterHTML.Set(pdfg.footerHTMLPath)
	}

	// Apply global replacements if not already set on page
	if pdfg.replace.value != nil {
		if opts.Replace.value == nil {
			opts.Replace.value = make(map[string]string)
		}
		for k, v := range pdfg.replace.value {
			if _, exists := opts.Replace.value[k]; !exists {
				opts.Replace.value[k] = v
			}
		}
	}

	pdfg.pages = append(pdfg.pages, p)
}

// SetPages resets all pages
func (pdfg *PDFGenerator) SetPages(p []PageProvider) {
	pdfg.pages = p
}

// ResetPages drops all pages previously added by AddPage or SetPages.
// This allows reuse of current instance of PDFGenerator with all of it's configuration preserved.
func (pdfg *PDFGenerator) ResetPages() {
	pdfg.pages = []PageProvider{}
}

// Buffer returns the embedded output buffer used if OutputFile is empty
func (pdfg *PDFGenerator) Buffer() *bytes.Buffer {
	return &pdfg.outbuf
}

// Bytes returns the output byte slice from the output buffer used if OutputFile is empty
func (pdfg *PDFGenerator) Bytes() []byte {
	return pdfg.outbuf.Bytes()
}

// SetOutput sets the output to write the PDF to, when this method is called, the internal buffer will not be used,
// so the Bytes(), Buffer() and WriteFile() methods will not work.
func (pdfg *PDFGenerator) SetOutput(w io.Writer) {
	pdfg.outWriter = w
}

// SetStderr sets the output writer for Stderr when running the wkhtmltopdf command. You only need to call this when you
// want to print the output of wkhtmltopdf (like the progress messages in verbose mode). If not called, or if w is nil, the
// output of Stderr is kept in an internal buffer and returned as error message if there was an error when calling wkhtmltopdf.
func (pdfg *PDFGenerator) SetStderr(w io.Writer) {
	pdfg.stdErr = w
}

// SetUserStyleSheet sets a global CSS stylesheet path to be applied to all subsequent pages added via AddPage.
// This setting overrides any UserStyleSheet setting on individual PageOptions unless the path is empty.
// It corresponds to the --user-style-sheet wkhtmltopdf option.
func (pdfg *PDFGenerator) SetUserStyleSheet(path string) {
	pdfg.userStyleSheetPath = path
}

// SetHeaderHTML sets a global header HTML file path to be applied to all subsequent pages added via AddPage.
// This setting overrides any HeaderHTML setting on individual PageOptions unless the path is empty.
// It corresponds to the --header-html wkhtmltopdf option.
func (pdfg *PDFGenerator) SetHeaderHTML(path string) {
	pdfg.headerHTMLPath = path
}

// SetFooterHTML sets a global footer HTML file path to be applied to all subsequent pages added via AddPage.
// This setting overrides any FooterHTML setting on individual PageOptions unless the path is empty.
// It corresponds to the --footer-html wkhtmltopdf option.
func (pdfg *PDFGenerator) SetFooterHTML(path string) {
	pdfg.footerHTMLPath = path
}

// SetReplace adds a key-value pair for replacement in headers and footers (e.g., [date], [page], [author]).
// These replacements are applied globally to pages added after this call, unless a replacement
// with the same key is already defined specifically for a page.
// It corresponds to the --replace wkhtmltopdf option.
func (pdfg *PDFGenerator) SetReplace(key, value string) {
	pdfg.replace.Set(key, value)
}

// SetCover sets the cover page from an HTML file path.
// Options for the cover page (like zoom, margins) can be set directly via pdfg.Cover.pageOptions.
// It corresponds to the cover wkhtmltopdf command.
func (pdfg *PDFGenerator) SetCover(path string) {
	pdfg.Cover.Input = path
	// Note: Cover page options can be set directly via pdfg.Cover.pageOptions if needed.
}

// WriteFile writes the contents of the output buffer to a file
func (pdfg *PDFGenerator) WriteFile(filename string) error {
	return os.WriteFile(filename, pdfg.Bytes(), 0666)
}

var lookPath = exec.LookPath

// findPath finds the path to wkhtmltopdf by
// - first looking in the current dir
// - looking in the PATH and PATHEXT environment dirs
// - using the WKHTMLTOPDF_PATH environment dir
// Warning: Running executables from the current path is no longer possible in Go 1.19
// See https://pkg.go.dev/os/exec@master#hdr-Executables_in_the_current_directory
// The path is cached, meaning you can not change the location of wkhtmltopdf in
// a running program once it has been found
func (pdfg *PDFGenerator) findPath() error {
	const exe = "wkhtmltopdf"
	pdfg.binPath = GetPath()
	if pdfg.binPath != "" {
		// wkhtmltopdf has already been found, return
		return nil
	}
	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	path, err := lookPath(filepath.Join(exeDir, exe))
	if err == nil && path != "" {
		binPath.Set(path)
		pdfg.binPath = path
		return nil
	}
	path, err = lookPath(exe)
	if errors.Is(err, exec.ErrDot) {
		return err
	}
	if err == nil && path != "" {
		binPath.Set(path)
		pdfg.binPath = path
		return nil
	}
	dir := os.Getenv("WKHTMLTOPDF_PATH")
	if dir == "" {
		return fmt.Errorf("%s not found", exe)
	}
	path, err = lookPath(filepath.Join(dir, exe))
	if errors.Is(err, exec.ErrDot) {
		return err
	}
	if err == nil && path != "" {
		binPath.Set(path)
		pdfg.binPath = path
		return nil
	}
	return fmt.Errorf("%s not found", exe)
}

func (pdfg *PDFGenerator) checkDuplicateFlags() error {
	// we currently can only have duplicates in the global options, so we only check these
	var options []string
	for _, arg := range pdfg.globalOptions.Args() {
		if strings.HasPrefix(arg, "--") { // this is not ideal, the value could also have this prefix
			for _, option := range options {
				if option == arg {
					return fmt.Errorf("duplicate argument: %s", arg)
				}
			}
			options = append(options, arg)
		}
	}
	return nil
}

// Create creates the PDF document and stores it in the internal buffer if no error is returned
func (pdfg *PDFGenerator) Create() error {
	return pdfg.run(context.Background())
}

// CreateContext is Create with a context passed to exec.CommandContext when calling wkhtmltopdf
func (pdfg *PDFGenerator) CreateContext(ctx context.Context) error {
	return pdfg.run(ctx)
}

func (pdfg *PDFGenerator) run(ctx context.Context) error {
	// check for duplicate flags
	err := pdfg.checkDuplicateFlags()
	if err != nil {
		return err
	}

	// create command
	cmd := exec.CommandContext(ctx, pdfg.binPath, pdfg.Args()...)

	// configure the commande (different for each OS, windows only for now (hides the cmd console))
	cmdConfig(cmd)

	// set stderr to the provided writer, or create a new buffer
	var errBuf *bytes.Buffer
	cmd.Stderr = pdfg.stdErr
	if cmd.Stderr == nil {
		errBuf = new(bytes.Buffer)
		cmd.Stderr = errBuf
	}

	// set output to the desired writer or the internal buffer
	if pdfg.outWriter != nil {
		cmd.Stdout = pdfg.outWriter
	} else {
		pdfg.outbuf.Reset() // reset internal buffer when we use it
		cmd.Stdout = &pdfg.outbuf
	}

	// if there is a pageReader page (from Stdin) we set Stdin to that reader
	for _, page := range pdfg.pages {
		if page.Reader() != nil {
			cmd.Stdin = page.Reader()
			break
		}
	}

	// run cmd to create the PDF
	err = cmd.Run()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		// on an error, return the error and the contents of Stderr if it was our own buffer
		// if Stderr was set to a custom writer, just return err
		if errBuf != nil {
			if errStr := errBuf.String(); strings.TrimSpace(errStr) != "" {
				return fmt.Errorf("%s\n%s", errStr, err)
			}
		}
		return err
	}
	return nil
}

// NewPDFGenerator returns a new PDFGenerator struct with all options created and
// checks if wkhtmltopdf can be found on the system
func NewPDFGenerator() (*PDFGenerator, error) {
	pdfg := NewPDFPreparer()
	return pdfg, pdfg.findPath()
}

// NewPDFPreparer returns a PDFGenerator object without looking for the wkhtmltopdf executable file.
// This is useful to prepare a PDF file that is generated elsewhere and you just want to save the config as JSON.
// Note that Create() can not be called on this object unless you call SetPath yourself.
func NewPDFPreparer() *PDFGenerator {
	return &PDFGenerator{
		globalOptions:  newGlobalOptions(),
		outlineOptions: newOutlineOptions(),
		Cover: cover{
			pageOptions: newPageOptions(),
		},
		TOC: toc{
			allTocOptions: allTocOptions{
				tocOptions:             newTocOptions(),
				pageOptions:            newPageOptions(),
				headerAndFooterOptions: newHeaderAndFooterOptions(),
			},
		},
		userStyleSheetPath: "", // Initialize new fields
		headerHTMLPath:     "",
		footerHTMLPath:     "",
		replace:            mapOption{option: "replace"}, // Initialize replace map
	}
}
