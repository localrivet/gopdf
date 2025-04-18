package wkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPDFGenerator(tb testing.TB) *PDFGenerator {
	pdfg, err := NewPDFGenerator()
	if err != nil {
		tb.Fatal(err)
	}

	pdfg.Dpi.Set(600)
	pdfg.NoCollate.Set(false)
	pdfg.PageSize.Set(PageSizeA4)
	pdfg.MarginBottom.Set(40)
	pdfg.MarginLeft.Set(0)

	page1 := NewPage("https://www.google.com")

	page1.DisableSmartShrinking.Set(true)
	page1.HeaderSpacing.Set(10.01)
	page1.Allow.Set("/usr/local/html")
	page1.Allow.Set("/usr/local/images")
	page1.CustomHeader.Set("X-AppKey", "abcdef")
	page1.ViewportSize.Set("3840x2160")
	page1.EnableLocalFileAccess.Set(true)

	pdfg.AddPage(page1)

	pdfg.Cover.Input = "https://wkhtmltopdf.org/index.html"
	pdfg.Cover.Zoom.Set(0.75)

	pdfg.TOC.Include = true
	pdfg.TOC.DisableDottedLines.Set(true)

	return pdfg
}

func expectedArgString() string {
	return "--dpi 600 --margin-bottom 40 --margin-left 0 --page-size A4 cover https://wkhtmltopdf.org/index.html --zoom 0.750 toc --disable-dotted-lines page https://www.google.com --allow /usr/local/html --allow /usr/local/images --custom-header X-AppKey abcdef --disable-smart-shrinking --enable-local-file-access --viewport-size 3840x2160 --header-spacing 10.010 -"
}

func TestArgString(t *testing.T) {
	pdfg := newTestPDFGenerator(t)
	assert.Equal(t, expectedArgString(), pdfg.ArgString())

	pdfg.SetPages(pdfg.pages)
	assert.Equal(t, expectedArgString(), pdfg.ArgString())
}

func TestResetPages(t *testing.T) {
	//Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}

	// Add 2 pages
	pdfg.AddPage(NewPage("https://www.google.com"))
	pdfg.AddPage(NewPage("https://www.github.com"))

	// check that we have two pages
	if len(pdfg.pages) != 2 {
		t.Errorf("Want 2 pages, have %d", len(pdfg.pages))
	}

	// Reset
	pdfg.ResetPages()

	// check that we have no pages
	if len(pdfg.pages) != 0 {
		t.Errorf("Want 0 pages, have %d", len(pdfg.pages))
	}
}

func TestVersion(t *testing.T) {
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}
	pdfg.Version.Set(true)
	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoInput(t *testing.T) {
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}
	err = pdfg.Create()
	if err == nil {
		t.Fatal("Want an error when there is no input, have no error")
	}
	//TODO temp error check because older versions of wkhtmltopdf return a different error :(
	wantErrNew := "You need to specify at least one input file, and exactly one output file"
	wantErrOld := "You need to specify atleast one input file, and exactly one output file"
	if strings.HasPrefix(err.Error(), wantErrNew) == false && strings.HasPrefix(err.Error(), wantErrOld) == false {
		t.Errorf("Want error prefix %s or %s, have %s", wantErrNew, wantErrOld, err.Error())
	}
}

func TestGeneratePDF(t *testing.T) {
	pdfg := newTestPDFGenerator(t)
	err := pdfg.Create()
	require.NoError(t, err)

	err = pdfg.WriteFile("testdata/TestGeneratePDF.pdf")
	require.NoError(t, err)

	t.Logf("PDF size %vkB", len(pdfg.Bytes())/1024)
}

func TestContextCancellation(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "windows" {
		t.Skip("temporarily skipping on Windows Github actions, because it blocks. Most likely on due to WindowStatus being set, need to investigate")
	}

	pdfg := newTestPDFGenerator(t)
	htmlfile, err := os.ReadFile("testdata/htmlsimple.html")
	if err != nil {
		t.Fatal(err)
	}

	htmlPage := NewPageReader(bytes.NewReader(htmlfile))
	// WindowStatus waits for page window status to be set to specified value,
	// only then it'd render PDF. In our case, it'd never happen and context
	// cancellation should cancel the request.
	htmlPage.WindowStatus.Set("dummy-status")
	ctx, cancelFunc := context.WithTimeout(context.TODO(), 200*time.Millisecond)
	defer cancelFunc()

	pdfg.AddPage(htmlPage)

	errBuf := new(bytes.Buffer)
	pdfg.SetStderr(errBuf)
	err = pdfg.CreateContext(ctx)
	if err == nil || err.Error() != "context deadline exceeded" {
		t.Errorf("Error should be `context deadline exceeded` but is `%v`", err)
	}
}

func TestGeneratePdfFromStdinSimple(t *testing.T) {
	//Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}
	htmlfile, err := os.ReadFile("testdata/htmlsimple.html")
	if err != nil {
		t.Fatal(err)
	}
	pdfg.AddPage(NewPageReader(bytes.NewReader(htmlfile)))
	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}
	err = pdfg.WriteFile("testdata/TestGeneratePdfFromStdinSimple.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("PDF size %vkB", len(pdfg.Bytes())/1024)
	if pdfg.Buffer().Len() != len(pdfg.Bytes()) {
		t.Errorf("Buffersize not equal")
	}
}

func TestPDFGeneratorOutputFile(t *testing.T) {
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}
	htmlfile, err := os.Open("testdata/htmlsimple.html")
	if err != nil {
		t.Fatal(err)
	}
	defer htmlfile.Close()

	pdfg.OutputFile = "testdata/TestPDFGeneratorOutputFile.pdf"

	pdfg.AddPage(NewPageReader(htmlfile))
	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}

	pdfFile, err := os.Open("testdata/TestPDFGeneratorOutputFile.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer pdfFile.Close()

	stat, err := pdfFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() < 100 {
		t.Errorf("generated PDF is size under 100 bytes")
	}
}

func TestGeneratePdfFromStdinHtml5(t *testing.T) {
	//Use newTestPDFGenerator and append to page1 and TOC
	pdfg := newTestPDFGenerator(t)
	htmlfile, err := os.ReadFile("testdata/html5.html")
	if err != nil {
		t.Fatal(err)
	}

	page2 := NewPageReader(bytes.NewReader(htmlfile))
	page2.EnableLocalFileAccess.Set(true)
	pdfg.AddPage(page2)

	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}
	err = pdfg.WriteFile("testdata/TestGeneratePdfFromStdinHtml5.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("PDF size %vkB", len(pdfg.Bytes())/1024)
}

func TestSetFooter(t *testing.T) {
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}

	p1 := NewPage("https://www.google.com")
	p1.FooterRight.Set("This is page [page]")
	p1.FooterFontSize.Set(10)

	pdfg.AddPage(p1)

	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}
	err = pdfg.WriteFile("testdata/TestSetFooter.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("PDF size %vkB", len(pdfg.Bytes())/1024)
}

func TestPath(t *testing.T) {
	path := "/usr/wkhtmltopdf/wkhtmltopdf"
	SetPath(path)
	defer SetPath("")
	if GetPath() != path {
		t.Errorf("Have path %q, want %q", GetPath(), path)
	}
}

func TestPDFGenerator_SetOutput(t *testing.T) {
	//Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}

	htmlfile, err := os.Open("testdata/htmlsimple.html")
	if err != nil {
		t.Fatal(err)
	}
	defer htmlfile.Close()

	pdfg.AddPage(NewPageReader(htmlfile))

	outBuf := new(bytes.Buffer)
	pdfg.SetOutput(outBuf)

	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}

	b := pdfg.Bytes()
	if len(b) != 0 {
		t.Errorf("expected to have zero bytes in internal buffer, have %d", len(b))
	}

	b = outBuf.Bytes()
	if len(b) < 3000 {
		t.Errorf("expected to have > 3000 bytes in output buffer, have %d", len(b))
	}
}

func TestPDFGenerator_SetStderr(t *testing.T) {
	//Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}

	htmlfile, err := os.Open("testdata/htmlsimple.html")
	if err != nil {
		t.Fatal(err)
	}
	defer htmlfile.Close()

	pdfg.AddPage(NewPageReader(htmlfile))

	errBuf := new(bytes.Buffer)
	pdfg.SetStderr(errBuf)

	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}

	// not sure if this is correct for all versions of wkhtmltopdf and if it is always in English
	outputStr := errBuf.String()
	shouldContain := []string{"Loading pages", "Printing pages"}
	for _, s := range shouldContain {
		if strings.Contains(outputStr, s) == false {
			t.Errorf("Stderr should contain %q, but it does not", s)
		}
	}
}

func TestTOCAndCustomFooter(t *testing.T) {
	//Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}

	htmlfile, err := os.Open("testdata/html5.html")
	if err != nil {
		t.Fatal(err)
	}
	defer htmlfile.Close()
	page := NewPageReader(htmlfile)
	page.EnableLocalFileAccess.Set(true) //needed to include js
	pdfg.AddPage(page)

	page.FooterHTML.Set("testdata/footer.html")
	page.FooterSpacing.Set(8)

	pdfg.TOC.XslStyleSheet.Set("testdata/toc.xls")
	pdfg.TOC.Include = true
	pdfg.TOC.EnableLocalFileAccess.Set(true) //needed to include js
	pdfg.TOC.FooterHTML.Set("testdata/footer-toc.html")
	pdfg.TOC.FooterSpacing.Set(8)

	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}

	// Write buffer contents to file on disk
	err = pdfg.WriteFile("testdata/TestGeneratePdfTOCAndCustomFooter.pdf")
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnitOptions(t *testing.T) {
	//Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	assert.NoError(t, err)

	// Add a page
	pdfg.AddPage(NewPage("https://www.google.com"))

	// Set all unit options
	pdfg.MarginRightUnit.Set("1mm")
	pdfg.MarginLeftUnit.Set("2cm")
	pdfg.MarginBottomUnit.Set("0.5cm")
	pdfg.MarginTopUnit.Set("10mm")
	pdfg.PageHeightUnit.Set("10in")
	pdfg.PageWidthUnit.Set("5.5in")

	want := `--margin-bottom 0.5cm --margin-left 2cm --margin-right 1mm --margin-top 10mm --page-height 10in --page-width 5.5in page https://www.google.com -`
	assert.Equal(t, want, pdfg.ArgString())

	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}

	// Write buffer contents to file on disk
	err = pdfg.WriteFile("testdata/TestUnitOptions.pdf")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateOptions(t *testing.T) {
	//Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	assert.NoError(t, err)

	// Add a page
	pdfg.AddPage(NewPage("https://www.google.com"))

	// Set a duplicate option (the value can be different)
	pdfg.MarginRight.Set(1)
	pdfg.MarginRightUnit.Set("1cm")

	err = pdfg.Create()
	assert.EqualError(t, err, "duplicate argument: --margin-right")
}

func TestBufferReset(t *testing.T) {
	// Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	if err != nil {
		t.Fatal(err)
	}

	// Add one page
	htmlfile, err := os.ReadFile("testdata/htmlsimple.html")
	if err != nil {
		t.Fatal(err)
	}
	pdfg.AddPage(NewPageReader(bytes.NewReader(htmlfile)))

	// Create PDF
	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}

	// Test if the internal buffer is not emty
	bufSize := pdfg.Buffer().Len()
	assert.Greater(t, bufSize, 0)

	// Reset pages and run same Create() again
	pdfg.ResetPages()
	pdfg.AddPage(NewPageReader(bytes.NewReader(htmlfile)))
	err = pdfg.Create()
	if err != nil {
		t.Fatal(err)
	}

	// Test is Buffer Size is equal to size of previous Create()
	assert.Equal(t, bufSize, pdfg.Buffer().Len())
}

func TestFindPath(t *testing.T) {
	defer func() { lookPath = exec.LookPath }()

	pdfgen := new(PDFGenerator)

	// lookpath finds a result immediately
	lookPath = func(file string) (string, error) {
		return file, nil
	}

	binPath.Set("")
	err := pdfgen.findPath()
	assert.NoError(t, err)

	// lookpath only returns a path when called with "wkhtmltopdf"
	lookPath = func(file string) (string, error) {
		if file == "wkhtmltopdf" {
			return file, nil
		}
		return "", errors.New("mock error")
	}

	binPath.Set("")
	err = pdfgen.findPath()
	assert.NoError(t, err)
	assert.Equal(t, "wkhtmltopdf", binPath.Get())

	// lookpath returns exec.ErrDot when called with "wkhtmltopdf"
	lookPath = func(file string) (string, error) {
		if file == "wkhtmltopdf" {
			return "", exec.ErrDot
		}
		return "", errors.New("mock error")
	}

	binPath.Set("")
	err = pdfgen.findPath()
	assert.Error(t, err)
	assert.EqualError(t, err, exec.ErrDot.Error())

	// lookpath only finds a path when "WKHTMLTOPDF_PATH" is included
	const WKHTMLTOPDFPATH = "/fake/path"
	os.Setenv("WKHTMLTOPDF_PATH", WKHTMLTOPDFPATH)
	lookPath = func(file string) (string, error) {
		if strings.HasPrefix(file, filepath.Clean(WKHTMLTOPDFPATH)) {
			return file, nil
		}
		return "", errors.New("mock error")
	}

	binPath.Set("")
	err = pdfgen.findPath()
	assert.NoError(t, err)

	// lookpath returns exec.ErrDot when "WKHTMLTOPDF_PATH" is included
	lookPath = func(file string) (string, error) {
		if strings.HasPrefix(file, filepath.Clean(WKHTMLTOPDFPATH)) {
			return "", exec.ErrDot
		}
		return "", errors.New("mock error")
	}

	binPath.Set("")
	err = pdfgen.findPath()
	assert.Error(t, err)
	assert.EqualError(t, err, exec.ErrDot.Error())

	// lookpath always returns an error and WKHTMLTOPDF_PATH is empty
	os.Setenv("WKHTMLTOPDF_PATH", "")
	lookPath = func(file string) (string, error) {
		return "", errors.New("mock error")
	}

	binPath.Set("")
	err = pdfgen.findPath()
	assert.Error(t, err)
	assert.EqualError(t, err, "wkhtmltopdf not found")

	// lookpath always returns an error and WKHTMLTOPDF_PATH is NOT empty
	os.Setenv("WKHTMLTOPDF_PATH", WKHTMLTOPDFPATH)
	lookPath = func(file string) (string, error) {
		return "", errors.New("mock error")
	}

	binPath.Set("")
	err = pdfgen.findPath()
	assert.Error(t, err)
	assert.EqualError(t, err, "wkhtmltopdf not found")
}

func TestStringOption(t *testing.T) {
	opt := stringOption{
		option: "stringopt",
	}
	opt.Set("value123")

	want := []string{"--stringopt", "value123"}

	if !reflect.DeepEqual(opt.Parse(), want) {
		t.Errorf("expected %v, have %v", want, opt.Parse())
	}

	opt.Unset()
	if !reflect.DeepEqual(opt.Parse(), []string{}) {
		t.Errorf("not empty after unset")
	}
}

func TestSliceOption(t *testing.T) {
	opt := sliceOption{
		option: "sliceopt",
	}
	opt.Set("string15183")
	opt.Set("foo")
	opt.Set("bar")

	want := []string{"--sliceopt", "string15183", "--sliceopt", "foo", "--sliceopt", "bar"}

	if !reflect.DeepEqual(opt.Parse(), want) {
		t.Errorf("expected %v, have %v", want, opt.Parse())
	}

	opt.Unset()
	if !reflect.DeepEqual(opt.Parse(), []string{}) {
		t.Errorf("not empty after unset")
	}
}

func TestMapOption(t *testing.T) {
	opt := mapOption{
		option: "mapopt",
	}

	opt.Set("key1", "foo")
	opt.Set("key2", "bar")
	opt.Set("key3", "Hello")

	result := strings.Join(opt.Parse(), " ")
	if !strings.Contains(result, "--mapopt key1 foo") {
		t.Error("missing map option key1")
	}
	if !strings.Contains(result, "--mapopt key2 bar") {
		t.Error("missing map option key2")
	}
	if !strings.Contains(result, "--mapopt key3 Hello") {
		t.Error("missing map option key3")
	}

	opt.Unset()
	if !reflect.DeepEqual(opt.Parse(), []string{}) {
		t.Errorf("not empty after unset")
	}
}

func TestUIntOption(t *testing.T) {
	opt := uintOption{
		option: "uintopt",
	}
	opt.Set(14860)

	want := []string{"--uintopt", "14860"}

	if !reflect.DeepEqual(opt.Parse(), want) {
		t.Errorf("expected %v, have %v", want, opt.Parse())
	}

	opt.Unset()
	if !reflect.DeepEqual(opt.Parse(), []string{}) {
		t.Errorf("not empty after unset")
	}
}

func TestFloatOption(t *testing.T) {
	opt := floatOption{
		option: "flopt",
	}
	opt.Set(239.75)

	want := []string{"--flopt", "239.750"}

	if !reflect.DeepEqual(opt.Parse(), want) {
		t.Errorf("expected %v, have %v", want, opt.Parse())
	}

	opt.Unset()
	if !reflect.DeepEqual(opt.Parse(), []string{}) {
		t.Errorf("not empty after unset")
	}
}

func TestBoolOption(t *testing.T) {
	opt := boolOption{
		option: "boolopt",
	}
	opt.Set(true)

	want := []string{"--boolopt"}

	if !reflect.DeepEqual(opt.Parse(), want) {
		t.Errorf("expected %v, have %v", want, opt.Parse())
	}

	opt.Unset()
	if !reflect.DeepEqual(opt.Parse(), []string{}) {
		t.Errorf("not empty after unset")
	}
}

func BenchmarkArgs(b *testing.B) {
	pdfg := newTestPDFGenerator(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pdfg.Args()
	}
}

func TestMarkdownPage(t *testing.T) {
	// Use a new blank PDF generator
	pdfg, err := NewPDFGenerator()
	require.NoError(t, err, "Failed to create PDFGenerator")

	// Create and add a markdown page
	mdPage := NewMarkdownPage("testdata/testmd.md")
	pdfg.AddPage(mdPage)

	// Create the PDF
	err = pdfg.Create()
	require.NoError(t, err, "Failed to create PDF from Markdown")

	// Check the output buffer
	pdfBytes := pdfg.Bytes()
	assert.NotEmpty(t, pdfBytes, "PDF output buffer should not be empty")

	// Check for PDF magic number
	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")), "Output does not start with PDF magic number")

	// Optional: Write to file for manual inspection
	// err = pdfg.WriteFile("testdata/TestMarkdownPage.pdf")
	// require.NoError(t, err, "Failed to write test PDF file")

	t.Logf("Markdown PDF size %vkB", len(pdfBytes)/1024)
}
