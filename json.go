package wkhtmltopdf

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

type jsonPDFGenerator struct {
	GlobalOptions  globalOptions
	OutlineOptions outlineOptions
	Cover          cover
	TOC            toc
	Pages          []jsonPage
}

type jsonPage struct {
	Type           string // "page", "reader", or "markdown"
	PageOptions    PageOptions
	InputFile      string // URL/Path for Page, "-" for Reader/Markdown
	InputPath      string // Path for MarkdownPage
	Base64PageData string // Base64 content for Reader/Markdown
}

// ToJSON creates JSON of the complete representation of the PDFGenerator.
// It also saves all pages. For a PageReader page, the content is stored as a Base64 string in the JSON.
func (pdfg *PDFGenerator) ToJSON() ([]byte, error) {

	jpdf := &jsonPDFGenerator{
		TOC:            pdfg.TOC,
		Cover:          pdfg.Cover,
		GlobalOptions:  pdfg.globalOptions,
		OutlineOptions: pdfg.outlineOptions,
	}

	for _, p := range pdfg.pages {
		jp := jsonPage{
			InputFile: p.InputFile(), // Get InputFile value ("-" or path/URL)
		}
		var pageContentReader io.Reader // To store reader for Base64 encoding if needed

		switch tp := p.(type) {
		case *Page:
			jp.Type = "page"
			jp.PageOptions = *tp.Options() // Use Options() method
			// No Base64 data needed for Page type
		case *PageReader:
			jp.Type = "reader"
			jp.PageOptions = *tp.Options()
			pageContentReader = tp.Reader() // Get the reader for Base64 encoding
		case *MarkdownPage:
			jp.Type = "markdown"
			jp.PageOptions = *tp.Options()
			jp.InputPath = tp.InputPath     // Store original Markdown path
			pageContentReader = tp.Reader() // Get the reader (provides converted HTML) for Base64 encoding
		default:
			// Should not happen if all PageProvider types are handled
			return nil, fmt.Errorf("unknown PageProvider type encountered during JSON serialization: %T", p)
		}

		// If it's a type that provides content via Reader (PageReader or MarkdownPage)
		if pageContentReader != nil {
			buf, err := io.ReadAll(pageContentReader)
			if err != nil {
				// Check if it's our specific errorReader from MarkdownPage conversion failure
				if er, ok := pageContentReader.(*errorReader); ok {
					return nil, fmt.Errorf("error reading content for JSON serialization (from %s): %w", jp.Type, er.err)
				}
				return nil, fmt.Errorf("error reading content for JSON serialization (from %s): %w", jp.Type, err)
			}
			jp.Base64PageData = base64.StdEncoding.EncodeToString(buf)
		}

		jpdf.Pages = append(jpdf.Pages, jp)
	}
	return json.Marshal(jpdf)
}

// NewPDFGeneratorFromJSON creates a new PDFGenerator and restores all the settings and pages
// from a JSON byte slice which should be created using PDFGenerator.ToJSON().
func NewPDFGeneratorFromJSON(jsonReader io.Reader) (*PDFGenerator, error) {

	jp := new(jsonPDFGenerator)

	err := json.NewDecoder(jsonReader).Decode(jp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %s", err)
	}

	pdfg, err := NewPDFGenerator()
	if err != nil {
		return nil, fmt.Errorf("error creating PDF generator: %s", err)
	}

	pdfg.TOC = jp.TOC
	pdfg.Cover = jp.Cover
	pdfg.globalOptions = jp.GlobalOptions
	pdfg.outlineOptions = jp.OutlineOptions

	for i, p := range jp.Pages {
		switch p.Type {
		case "page":
			// InputFile should contain the URL or path
			if p.InputFile == "" || p.InputFile == "-" {
				return nil, fmt.Errorf("invalid InputFile value for page type on page %d", i)
			}
			page := NewPage(p.InputFile)
			page.PageOptions = p.PageOptions // Restore options
			pdfg.AddPage(page)

		case "reader":
			// Content should be in Base64PageData
			if p.Base64PageData == "" {
				return nil, fmt.Errorf("missing Base64PageData for reader type on page %d", i)
			}
			buf, err := base64.StdEncoding.DecodeString(p.Base64PageData)
			if err != nil {
				return nil, fmt.Errorf("error decoding base64 input for reader type on page %d: %w", i, err)
			}
			pageReader := NewPageReader(bytes.NewReader(buf))
			pageReader.PageOptions = p.PageOptions // Restore options
			pdfg.AddPage(pageReader)

		case "markdown":
			// InputPath should contain the original Markdown file path
			if p.InputPath == "" {
				return nil, fmt.Errorf("missing InputPath for markdown type on page %d", i)
			}
			// Recreate MarkdownPage from the path; it will handle reading/conversion
			markdownPage := NewMarkdownPage(p.InputPath)
			markdownPage.PageOptions = p.PageOptions // Restore options
			pdfg.AddPage(markdownPage)
			// Note: We ignore Base64PageData here, relying on InputPath for Markdown

		default:
			return nil, fmt.Errorf("unknown page type %q encountered during JSON deserialization on page %d", p.Type, i)
		}
	}

	return pdfg, nil
}

type jsonBoolOption struct {
	Option string
	Value  bool
}

func (bo *boolOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonBoolOption{bo.option, bo.value})
}

func (bo *boolOption) UnmarshalJSON(b []byte) error {
	jbo := new(jsonBoolOption)
	err := json.Unmarshal(b, jbo)
	if err != nil {
		return err
	}
	bo.value = jbo.Value
	bo.option = jbo.Option
	return nil
}

type jsonStringOption struct {
	Option string
	Value  string
}

func (so *stringOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonStringOption{so.option, so.value})
}

func (so *stringOption) UnmarshalJSON(b []byte) error {
	jso := new(jsonStringOption)
	err := json.Unmarshal(b, jso)
	if err != nil {
		return err
	}
	so.value = jso.Value
	so.option = jso.Option
	return nil
}

type jsonUintOption struct {
	Option string
	IsSet  bool
	Value  uint
}

func (io *uintOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonUintOption{io.option, io.isSet, io.value})
}

func (io *uintOption) UnmarshalJSON(b []byte) error {
	jio := new(jsonUintOption)
	err := json.Unmarshal(b, jio)
	if err != nil {
		return err
	}
	io.value = jio.Value
	io.isSet = jio.IsSet
	io.option = jio.Option
	return nil
}

type jsonFloatOption struct {
	Option string
	IsSet  bool
	Value  float64
}

func (fo *floatOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonFloatOption{fo.option, fo.isSet, fo.value})
}

func (fo *floatOption) UnmarshalJSON(b []byte) error {
	jfo := new(jsonFloatOption)
	err := json.Unmarshal(b, jfo)
	if err != nil {
		return err
	}
	fo.value = jfo.Value
	fo.isSet = jfo.IsSet
	fo.option = jfo.Option
	return nil
}

type jsonMapOption struct {
	Option string
	Value  map[string]string
}

func (mo *mapOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonMapOption{mo.option, mo.value})
}

func (mo *mapOption) UnmarshalJSON(b []byte) error {
	jmo := new(jsonMapOption)
	err := json.Unmarshal(b, jmo)
	if err != nil {
		return err
	}
	mo.value = jmo.Value
	mo.option = jmo.Option
	return nil
}

type jsonSliceOption struct {
	Option string
	Value  []string
}

func (so *sliceOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonSliceOption{so.option, so.value})
}

func (so *sliceOption) UnmarshalJSON(b []byte) error {
	jso := new(jsonSliceOption)
	err := json.Unmarshal(b, jso)
	if err != nil {
		return err
	}
	so.value = jso.Value
	so.option = jso.Option
	return nil
}
