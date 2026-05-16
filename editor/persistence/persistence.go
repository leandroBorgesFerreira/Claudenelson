package persistence

import (
	"encoding/json"
	"math/rand"
	"os"
	"time"

	"claudenelson/editor/block"
	"claudenelson/editor/document"
	"claudenelson/editor/factory"
	"claudenelson/editor/format"
)

// ========================================
// Writeopia-Compatible JSON Structures
// ========================================

// WriteopiaSpan represents inline formatting in Writeopia format
type WriteopiaSpan struct {
	Start int     `json:"start"`
	End   int     `json:"end"`
	Span  string  `json:"span"`
	Extra *string `json:"extra,omitempty"`
}

// WriteopiaTag represents block-level formatting (headers)
type WriteopiaTag struct {
	Tag      string `json:"tag"`
	Position int    `json:"position"`
}

// WriteopiaType represents the type of a StoryStep
type WriteopiaType struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
}

// WriteopiaStoryStep represents a content element in Writeopia format
type WriteopiaStoryStep struct {
	ID           string          `json:"id"`
	Type         WriteopiaType   `json:"type"`
	Text         string          `json:"text,omitempty"`
	Position     int             `json:"position"`
	Tags         []WriteopiaTag  `json:"tags,omitempty"`
	Spans        []WriteopiaSpan `json:"spans,omitempty"`
	Decoration   interface{}     `json:"decoration,omitempty"`
	Checked      *bool           `json:"checked,omitempty"`
	DocumentLink interface{}     `json:"documentLink,omitempty"`
}

// WriteopiaDocument represents a complete Writeopia document
type WriteopiaDocument struct {
	ID            string               `json:"id"`
	Title         string               `json:"title"`
	WorkspaceID   string               `json:"workspaceId"`
	Content       []WriteopiaStoryStep `json:"content"`
	CreatedAt     int64                `json:"createdAt"`
	LastUpdatedAt int64                `json:"lastUpdatedAt"`
	LastSyncedAt  *int64               `json:"lastSyncedAt,omitempty"`
	ParentID      string               `json:"parentId"`
	IsLocked      bool                 `json:"isLocked"`
	IsFavorite    bool                 `json:"isFavorite"`
	Deleted       bool                 `json:"deleted"`
	Icon          interface{}          `json:"icon,omitempty"`
}

// ========================================
// Legacy Structures (for backward compatibility)
// ========================================

// SerializedStyle represents text formatting in legacy JSON format
type SerializedStyle struct {
	Bold      bool `json:"bold,omitempty"`
	Italic    bool `json:"italic,omitempty"`
	Underline bool `json:"underline,omitempty"`
}

// SerializedSpan represents a formatting span in legacy JSON format
type SerializedSpan struct {
	Start int             `json:"start"`
	End   int             `json:"end"`
	Style SerializedStyle `json:"style"`
}

// SerializedBlock represents a block in legacy JSON format
type SerializedBlock struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Content string           `json:"content"`
	Spans   []SerializedSpan `json:"spans,omitempty"`
	Checked *bool            `json:"checked,omitempty"`
	Level   *int             `json:"level,omitempty"`
}

// SerializedDocument represents a document in legacy JSON format
type SerializedDocument struct {
	Blocks     []SerializedBlock `json:"blocks"`
	CursorLine int               `json:"cursorLine"`
	CursorCol  int               `json:"cursorCol"`
}

// ========================================
// Helper Functions
// ========================================

// generateID creates a random 10-character alphanumeric ID
func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 10)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// convertSpansToWriteopia converts format.Spans to Writeopia format
// In Writeopia, each style is a separate span entry
func convertSpansToWriteopia(spans format.Spans) []WriteopiaSpan {
	if len(spans) == 0 {
		return nil
	}
	var result []WriteopiaSpan
	for _, span := range spans {
		if span.Style.Bold {
			result = append(result, WriteopiaSpan{
				Start: span.Start,
				End:   span.End,
				Span:  "BOLD",
			})
		}
		if span.Style.Italic {
			result = append(result, WriteopiaSpan{
				Start: span.Start,
				End:   span.End,
				Span:  "ITALIC",
			})
		}
		if span.Style.Underline {
			result = append(result, WriteopiaSpan{
				Start: span.Start,
				End:   span.End,
				Span:  "UNDERLINE",
			})
		}
		if span.Style.Highlight {
			result = append(result, WriteopiaSpan{
				Start: span.Start,
				End:   span.End,
				Span:  "HIGHLIGHT",
			})
		}
	}
	return result
}

// convertWriteopiaSpansToFormat converts Writeopia spans to format.Spans
func convertWriteopiaSpansToFormat(wspans []WriteopiaSpan) format.Spans {
	if len(wspans) == 0 {
		return nil
	}
	// Group spans by position range using a struct key approach
	type posKey struct {
		start, end int
	}
	spanMap := make(map[posKey]*format.Span)
	for _, ws := range wspans {
		key := posKey{ws.Start, ws.End}
		if _, exists := spanMap[key]; !exists {
			spanMap[key] = &format.Span{Start: ws.Start, End: ws.End}
		}
		switch ws.Span {
		case "BOLD":
			spanMap[key].Style.Bold = true
		case "ITALIC":
			spanMap[key].Style.Italic = true
		case "UNDERLINE":
			spanMap[key].Style.Underline = true
		case "HIGHLIGHT", "HIGHLIGHT_GREEN", "HIGHLIGHT_RED":
			// Treat all highlight variants as highlight
			spanMap[key].Style.Highlight = true
		}
	}
	result := make(format.Spans, 0, len(spanMap))
	for _, span := range spanMap {
		result = append(result, *span)
	}
	return result
}

// blockTypeToWriteopia converts block type to Writeopia type and optional tags
func blockTypeToWriteopia(b block.Block, isTitle bool) (WriteopiaType, []WriteopiaTag) {
	if isTitle {
		return WriteopiaType{Name: "title", Number: 11}, nil
	}

	switch b.Type() {
	case block.TypeH1:
		return WriteopiaType{Name: "message", Number: 0}, []WriteopiaTag{{Tag: "H1", Position: 0}}
	case block.TypeH2:
		return WriteopiaType{Name: "message", Number: 0}, []WriteopiaTag{{Tag: "H2", Position: 0}}
	case block.TypeH3:
		return WriteopiaType{Name: "message", Number: 0}, []WriteopiaTag{{Tag: "H3", Position: 0}}
	case block.TypeH4:
		return WriteopiaType{Name: "message", Number: 0}, []WriteopiaTag{{Tag: "H4", Position: 0}}
	case block.TypeListItem:
		return WriteopiaType{Name: "unordered_list_item", Number: 16}, nil
	case block.TypeCheckboxItem:
		return WriteopiaType{Name: "check_item", Number: 10}, nil
	default:
		return WriteopiaType{Name: "message", Number: 0}, nil
	}
}

// extractTitle extracts the document title from blocks
func extractTitle(blocks []block.Block) string {
	if len(blocks) == 0 {
		return "Untitled"
	}
	// Use first block content as title
	return blocks[0].Content()
}

// ========================================
// Serialization (Document -> Writeopia JSON)
// ========================================

// serializeToWriteopia converts a document to Writeopia format
func serializeToWriteopia(doc *document.Document, docID string, createdAt int64) WriteopiaDocument {
	now := time.Now().UnixMilli()
	if createdAt == 0 {
		createdAt = now
	}

	title := extractTitle(doc.Blocks)

	content := make([]WriteopiaStoryStep, 0, len(doc.Blocks)+1)

	for i, b := range doc.Blocks {
		isTitle := i == 0
		wtype, tags := blockTypeToWriteopia(b, isTitle)

		step := WriteopiaStoryStep{
			ID:       b.ID(),
			Type:     wtype,
			Text:     b.Content(),
			Position: i,
			Tags:     tags,
			Spans:    convertSpansToWriteopia(b.Spans()),
		}

		// Handle checkbox checked state
		if cb, ok := b.(*block.CheckboxBlock); ok {
			checked := cb.IsChecked()
			step.Checked = &checked
		}

		content = append(content, step)
	}

	// Add empty message at end (Writeopia convention)
	content = append(content, WriteopiaStoryStep{
		ID:       generateID(),
		Type:     WriteopiaType{Name: "message", Number: 0},
		Text:     "",
		Position: len(doc.Blocks),
	})

	return WriteopiaDocument{
		ID:            docID,
		Title:         title,
		WorkspaceID:   "disconnected_user",
		Content:       content,
		CreatedAt:     createdAt,
		LastUpdatedAt: now,
		ParentID:      "root",
		IsLocked:      false,
		IsFavorite:    false,
		Deleted:       false,
	}
}

// Save saves the document to a Writeopia-compatible JSON file
func Save(doc *document.Document, path string) error {
	// Try to load existing document to preserve ID and createdAt
	var docID string
	var createdAt int64

	if existingData, err := os.ReadFile(path); err == nil {
		var existing WriteopiaDocument
		if json.Unmarshal(existingData, &existing) == nil && existing.ID != "" {
			docID = existing.ID
			createdAt = existing.CreatedAt
		}
	}

	if docID == "" {
		docID = generateID()
	}

	wdoc := serializeToWriteopia(doc, docID, createdAt)

	data, err := json.MarshalIndent(wdoc, "", "    ")
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

// ========================================
// Deserialization (JSON -> Document)
// ========================================

// deserializeSpans converts legacy serialized spans to format.Spans
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

// loadWriteopiaDocument loads a Writeopia format document
func loadWriteopiaDocument(data []byte, f *factory.BlockFactory) (*document.Document, error) {
	var wdoc WriteopiaDocument
	if err := json.Unmarshal(data, &wdoc); err != nil {
		return nil, err
	}

	doc := document.NewDocument()

	for i, step := range wdoc.Content {
		// Skip empty trailing message (Writeopia convention)
		if step.Text == "" && step.Type.Name == "message" && i == len(wdoc.Content)-1 {
			continue
		}

		var b block.Block
		spans := convertWriteopiaSpansToFormat(step.Spans)

		// Check for header tags
		hasH1 := false
		hasH2 := false
		hasH3 := false
		hasH4 := false
		for _, tag := range step.Tags {
			switch tag.Tag {
			case "H1":
				hasH1 = true
			case "H2":
				hasH2 = true
			case "H3":
				hasH3 = true
			case "H4":
				hasH4 = true
			}
		}

		switch step.Type.Name {
		case "title":
			// Title becomes H1
			b = f.CreateHeading(step.Text, 1)
		case "unordered_list_item":
			b = f.CreateListItemWithSpans(step.Text, spans)
		case "check_item":
			checked := false
			if step.Checked != nil {
				checked = *step.Checked
			}
			b = f.CreateCheckboxWithSpans(step.Text, checked, spans)
		case "message":
			if hasH1 {
				b = f.CreateHeading(step.Text, 1)
			} else if hasH2 {
				b = f.CreateHeading(step.Text, 2)
			} else if hasH3 {
				b = f.CreateHeading(step.Text, 3)
			} else if hasH4 {
				b = f.CreateHeading(step.Text, 4)
			} else {
				b = f.CreateTextWithSpans(step.Text, spans)
			}
		default:
			b = f.CreateTextWithSpans(step.Text, spans)
		}

		// Set spans for heading blocks
		if spans != nil {
			b.SetSpans(spans)
		}

		doc.AddBlock(b)
	}

	// Ensure at least one block
	if len(doc.Blocks) == 0 {
		doc.AddBlock(f.CreateText(""))
	}

	return doc, nil
}

// loadLegacyDocument loads a legacy format document
func loadLegacyDocument(data []byte, f *factory.BlockFactory) (*document.Document, error) {
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

		// Set spans for heading blocks
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

// isWriteopiaFormat checks if the JSON data is in Writeopia format
func isWriteopiaFormat(data []byte) bool {
	var probe struct {
		Content []interface{} `json:"content"`
		ID      string        `json:"id"`
		Title   string        `json:"title"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	// Writeopia format has "content" array and "id" field
	return probe.Content != nil && probe.ID != ""
}

// Load loads a document from a JSON file (supports both Writeopia and legacy formats)
func Load(path string, f *factory.BlockFactory) (*document.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if isWriteopiaFormat(data) {
		return loadWriteopiaDocument(data, f)
	}

	return loadLegacyDocument(data, f)
}
