package tool

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"encoding/json"
	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
)

// ReadDocument reads uploaded document files (TXT, DOCX, XLSX, PDF).
type ReadDocument struct{}

func (t *ReadDocument) Definition() Definition {
	return Definition{
		Name: "read_document",
		Description: `Read and extract text content from uploaded document files. Supports TXT, DOCX, XLSX, and PDF files.

Use this tool when the user uploads a document and you need to read its contents. The file_path should be the server-side path returned from the file upload response.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "Server-side file path (e.g., ./data/uploads/abc123_document.pdf)",
				},
				"file_type": map[string]any{
					"type":        "string",
					"enum":        []string{"txt", "docx", "xlsx", "pdf", "auto"},
					"description": "File type hint. Use 'auto' to detect from file extension.",
				},
			},
			"required": []string{"file_path"},
		},
	}
}

func (t *ReadDocument) Execute(call Call) (string, error) {
	var params struct {
		FilePath string `json:"file_path"`
		FileType string `json:"file_type"`
	}
	if err := decodeArgs(call.Arguments, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	fileType := params.FileType
	if fileType == "" || fileType == "auto" {
		fileType = detectTypeByExt(params.FilePath)
	}

	switch fileType {
	case "txt":
		return readTXT(params.FilePath)
	case "docx":
		return readDOCX(params.FilePath)
	case "xlsx":
		return readXLSX(params.FilePath)
	case "pdf":
		return readPDF(params.FilePath)
	default:
		return "", fmt.Errorf("unsupported file type: %s (supported: txt, docx, xlsx, pdf)", fileType)
	}
}

func detectTypeByExt(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".txt"), strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".csv"):
		return "txt"
	case strings.HasSuffix(lower, ".docx"):
		return "docx"
	case strings.HasSuffix(lower, ".xlsx"), strings.HasSuffix(lower, ".xls"):
		return "xlsx"
	case strings.HasSuffix(lower, ".pdf"):
		return "pdf"
	default:
		return "txt"
	}
}

func readTXT(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

func readDOCX(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("failed to open docx file: %w", err)
	}
	defer r.Close()

	var documentXML *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			documentXML = f
			break
		}
	}
	if documentXML == nil {
		return "", fmt.Errorf("word/document.xml not found in docx file")
	}

	rc, err := documentXML.Open()
	if err != nil {
		return "", fmt.Errorf("failed to read document.xml: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read document.xml: %w", err)
	}

	return extractTextFromDocxXML(data), nil
}

// docxXML represents the relevant parts of a DOCX document.xml.
type docxXML struct {
	XMLName xml.Name `xml:"document"`
	Body    docxBody `xml:"body"`
}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"p"`
	Tables     []docxTable     `xml:"tbl"`
}

type docxParagraph struct {
	Runs      []docxRun       `xml:"r"`
	Hyperlinks []docxHyperlink `xml:"hyperlink"`
}

// docxRunProps captures basic formatting flags from w:rPr.
type docxRunProps struct {
	Bold   bool `xml:"b"`
	Italic bool `xml:"i"`
}

type docxRun struct {
	RPr   docxRunProps `xml:"rPr"`
	Texts []docxText   `xml:"t"`
}

// docxHyperlink mirrors w:hyperlink which wraps runs (w:r) and may carry an
// anchor (internal link) or r:id (external relationship).
type docxHyperlink struct {
	Anchor string   `xml:"anchor,attr"`
	RId    string   `xml:"id,attr"`
	Runs   []docxRun `xml:"r"`
}

type docxText struct {
	Value string `xml:",chardata"`
}

type docxTable struct {
	Rows []docxRow `xml:"tr"`
}

type docxRow struct {
	Cells []docxCell `xml:"tc"`
}

type docxCell struct {
	Paragraphs []docxParagraph `xml:"p"`
}

func extractTextFromDocxXML(data []byte) string {
	var doc docxXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return string(data)
	}

	var result strings.Builder
	for _, p := range doc.Body.Paragraphs {
		var line strings.Builder
		for _, r := range p.Runs {
			line.WriteString(formatRunText(r))
		}
		// Process hyperlinks within the paragraph.
		for _, hl := range p.Hyperlinks {
			var hlText strings.Builder
			for _, r := range hl.Runs {
				hlText.WriteString(formatRunText(r))
			}
			text := strings.TrimSpace(hlText.String())
			if text == "" {
				continue
			}
			if hl.Anchor != "" {
				// Internal bookmark link.
				line.WriteString(fmt.Sprintf("[%s](#%s)", text, hl.Anchor))
			} else {
				// External link (r:id) — we don't resolve the relationship target
				// here, so just mark it as a link with its visible text.
				line.WriteString(fmt.Sprintf("[%s]", text))
			}
		}
		if line.Len() > 0 {
			result.WriteString(line.String())
			result.WriteString("\n")
		}
	}

	for _, tbl := range doc.Body.Tables {
		result.WriteString("\n[Table]\n")
		for _, row := range tbl.Rows {
			var cells []string
			for _, cell := range row.Cells {
				var cellText strings.Builder
				for _, p := range cell.Paragraphs {
					for _, r := range p.Runs {
						cellText.WriteString(formatRunText(r))
					}
					for _, hl := range p.Hyperlinks {
						for _, r := range hl.Runs {
							cellText.WriteString(formatRunText(r))
						}
					}
				}
				cells = append(cells, strings.TrimSpace(cellText.String()))
			}
			result.WriteString("| " + strings.Join(cells, " | ") + " |\n")
		}
		result.WriteString("\n")
	}

	return result.String()
}

// formatRunText extracts text from a run and applies markdown bold/italic
// markers based on the run's formatting properties (w:b / w:i).
func formatRunText(r docxRun) string {
	var text strings.Builder
	for _, t := range r.Texts {
		text.WriteString(t.Value)
	}
	s := text.String()
	if s == "" {
		return ""
	}
	if r.RPr.Bold && r.RPr.Italic {
		return "***" + s + "***"
	}
	if r.RPr.Bold {
		return "**" + s + "**"
	}
	if r.RPr.Italic {
		return "*" + s + "*"
	}
	return s
}

func readXLSX(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to open xlsx file: %w", err)
	}
	defer f.Close()

	var result strings.Builder
	sheets := f.GetSheetList()

	for _, sheet := range sheets {
		result.WriteString(fmt.Sprintf("\n=== %s ===\n", sheet))

		rows, err := f.GetRows(sheet)
		if err != nil {
			result.WriteString(fmt.Sprintf("(failed to read sheet: %v)\n", err))
			continue
		}

		if len(rows) == 0 {
			result.WriteString("(empty sheet)\n")
			continue
		}

		// Build a markdown-like table
		colWidths := make([]int, 0)
		for _, row := range rows {
			for len(colWidths) < len(row) {
				colWidths = append(colWidths, 0)
			}
			for i, cell := range row {
				for _, r := range []rune(cell) {
					w := 1
					if r > 127 {
						w = 2 // CJK characters take 2 display columns
					}
					colWidths[i] += w
				}
			}
		}

		for i, row := range rows {
			cells := make([]string, len(colWidths))
			for j := range cells {
				if j < len(row) {
					cells[j] = row[j]
				} else {
					cells[j] = ""
				}
			}
			result.WriteString("| " + strings.Join(cells, " | ") + " |\n")

			// Header separator after first row
			if i == 0 {
				seps := make([]string, len(colWidths))
				for j, w := range colWidths {
					if w < 3 {
						w = 3
					}
					seps[j] = strings.Repeat("-", w)
				}
				result.WriteString("|" + strings.Join(seps, "|") + "|\n")
			}
		}
	}

	return strings.TrimSpace(result.String()), nil
}

func readPDF(path string) (string, error) {
	f, reader, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer f.Close()

	plainText, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("failed to read PDF text: %w", err)
	}

	text, err := io.ReadAll(plainText)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF content: %w", err)
	}

	result := strings.TrimSpace(string(text))
	if result == "" {
		return "", fmt.Errorf("unable to extract text from this PDF. The PDF may be a scanned image — try using OCR tools instead.")
	}
	return result, nil
}

func decodeArgs(argsJSON string, v any) error {
	return json.Unmarshal([]byte(argsJSON), v)
}
