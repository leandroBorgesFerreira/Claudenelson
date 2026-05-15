package persistence

import (
	"encoding/json"
	"os"

	"claudenelson/editor/block"
	"claudenelson/editor/document"
	"claudenelson/editor/factory"
	"claudenelson/editor/format"
)

// SerializedStyle represents text formatting in JSON format
type SerializedStyle struct {
	Bold      bool `json:"bold,omitempty"`
	Italic    bool `json:"italic,omitempty"`
	Underline bool `json:"underline,omitempty"`
}

// SerializedSpan represents a formatting span in JSON format
type SerializedSpan struct {
	Start int             `json:"start"`
	End   int             `json:"end"`
	Style SerializedStyle `json:"style"`
}

// SerializedBlock represents a block in JSON format
type SerializedBlock struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Content string           `json:"content"`
	Spans   []SerializedSpan `json:"spans,omitempty"` // Formatting spans
	Checked *bool            `json:"checked,omitempty"` // CheckboxBlock
	Level   *int             `json:"level,omitempty"`   // HeadingBlock
}

// SerializedDocument represents a document in JSON format
type SerializedDocument struct {
	Blocks     []SerializedBlock `json:"blocks"`
	CursorLine int               `json:"cursorLine"`
	CursorCol  int               `json:"cursorCol"`
}

// serializeSpans converts format.Spans to serialized form
func serializeSpans(spans format.Spans) []SerializedSpan {
	if len(spans) == 0 {
		return nil
	}
	result := make([]SerializedSpan, len(spans))
	for i, span := range spans {
		result[i] = SerializedSpan{
			Start: span.Start,
			End:   span.End,
			Style: SerializedStyle{
				Bold:      span.Style.Bold,
				Italic:    span.Style.Italic,
				Underline: span.Style.Underline,
			},
		}
	}
	return result
}

// deserializeSpans converts serialized spans to format.Spans
func deserializeSpans(serialized []SerializedSpan) format.Spans {
	if len(serialized) == 0 {
		return nil
	}
	result := make(format.Spans, len(serialized))
	for i, ss := range serialized {
		result[i] = format.Span{
			Start: ss.Start,
			End:   ss.End,
			Style: format.Style{
				Bold:      ss.Style.Bold,
				Italic:    ss.Style.Italic,
				Underline: ss.Style.Underline,
			},
		}
	}
	return result
}

// serializeBlock converts a block to its serialized form
func serializeBlock(b block.Block) SerializedBlock {
	sb := SerializedBlock{
		ID:      b.ID(),
		Type:    b.Type().String(),
		Content: b.Content(),
		Spans:   serializeSpans(b.Spans()),
	}

	// Handle specialized block fields
	switch blk := b.(type) {
	case *block.CheckboxBlock:
		checked := blk.IsChecked()
		sb.Checked = &checked
	case *block.HeadingBlock:
		level := blk.Level()
		sb.Level = &level
	}

	return sb
}

// serialize converts a document to its serialized form
func serialize(doc *document.Document) SerializedDocument {
	blocks := make([]SerializedBlock, 0, len(doc.Blocks))
	for _, b := range doc.Blocks {
		blocks = append(blocks, serializeBlock(b))
	}

	return SerializedDocument{
		Blocks:     blocks,
		CursorLine: doc.CursorLine,
		CursorCol:  doc.CursorCol,
	}
}

// Save saves the document to a JSON file using atomic write
func Save(doc *document.Document, path string) error {
	data, err := json.MarshalIndent(serialize(doc), "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: write to temp file, then rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// Load loads a document from a JSON file
func Load(path string, f *factory.BlockFactory) (*document.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sd SerializedDocument
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, err
	}

	doc := document.NewDocument()

	for _, sb := range sd.Blocks {
		var b block.Block
		spans := deserializeSpans(sb.Spans)

		switch sb.Type {
		case "h1":
			b = f.CreateHeading(sb.Content, 1)
		case "h2":
			b = f.CreateHeading(sb.Content, 2)
		case "h3":
			b = f.CreateHeading(sb.Content, 3)
		case "h4":
			level := 4
			if sb.Level != nil {
				level = *sb.Level
			}
			b = f.CreateHeading(sb.Content, level)
		case "checkbox":
			checked := false
			if sb.Checked != nil {
				checked = *sb.Checked
			}
			b = f.CreateCheckboxWithSpans(sb.Content, checked, spans)
		case "list_item":
			b = f.CreateListItemWithSpans(sb.Content, spans)
		default:
			b = f.CreateTextWithSpans(sb.Content, spans)
		}

		// Set spans for heading blocks (they don't have WithSpans variant)
		if spans != nil {
			b.SetSpans(spans)
		}

		doc.AddBlock(b)
	}

	// Restore cursor position
	doc.CursorLine = sd.CursorLine
	doc.CursorCol = sd.CursorCol

	// Ensure cursor is within bounds
	if doc.CursorLine >= len(doc.Blocks) {
		doc.CursorLine = len(doc.Blocks) - 1
	}
	if doc.CursorLine < 0 {
		doc.CursorLine = 0
	}

	return doc, nil
}
