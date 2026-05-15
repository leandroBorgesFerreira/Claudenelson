package persistence

import (
	"encoding/json"
	"os"

	"claudenelson/editor/block"
	"claudenelson/editor/document"
	"claudenelson/editor/factory"
)

// SerializedBlock represents a block in JSON format
type SerializedBlock struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Checked *bool  `json:"checked,omitempty"` // CheckboxBlock
	Level   *int   `json:"level,omitempty"`   // HeadingBlock
}

// SerializedDocument represents a document in JSON format
type SerializedDocument struct {
	Blocks     []SerializedBlock `json:"blocks"`
	CursorLine int               `json:"cursorLine"`
	CursorCol  int               `json:"cursorCol"`
}

// serializeBlock converts a block to its serialized form
func serializeBlock(b block.Block) SerializedBlock {
	sb := SerializedBlock{
		ID:      b.ID(),
		Type:    b.Type().String(),
		Content: b.Content(),
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
			b = f.CreateCheckbox(sb.Content, checked)
		case "list_item":
			b = f.CreateListItem(sb.Content)
		default:
			b = f.CreateText(sb.Content)
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
