package editor

import (
	tea "github.com/charmbracelet/bubbletea"

	"claudenelson/editor/format"
)

// currentStyle returns the current formatting style based on active modes
func (m *Model) currentStyle() format.Style {
	return format.Style{
		Bold:      m.boldMode,
		Italic:    m.italicMode,
		Underline: m.underlineMode,
	}
}

// applyHighlightAndExit applies or removes highlight from the selected range and exits highlight mode
func (m *Model) applyHighlightAndExit() tea.Cmd {
	if !m.highlightMode {
		return nil
	}

	currentBlock := m.doc.CurrentBlock()
	if currentBlock != nil {
		start, end := m.selectionStart, m.doc.CursorCol
		if start > end {
			start, end = end, start
		}
		if start != end {
			// Toggle highlight span on current block
			spans := currentBlock.Spans()
			newSpans := spans.ToggleHighlight(start, end)
			currentBlock.SetSpans(newSpans)
		}
	}

	m.highlightMode = false
	return m.markDirty()
}

// exitHighlightMode exits highlight mode without applying
func (m *Model) exitHighlightMode() {
	m.highlightMode = false
}

// applyFormatToHandleSelectedLines applies a formatting style to all handle-selected lines
func (m *Model) applyFormatToHandleSelectedLines(style format.Style) tea.Cmd {
	if !m.hasHandleSelection() {
		return nil
	}

	for idx := range m.selectedLines {
		blk := m.doc.BlockAt(idx)
		if blk == nil {
			continue
		}
		content := blk.Content()
		contentLen := len([]rune(content))
		if contentLen == 0 {
			continue
		}

		// Apply style to entire line
		spans := blk.Spans()
		newSpan := format.Span{
			Start: 0,
			End:   contentLen,
			Style: style,
		}
		spans = append(spans, newSpan)
		spans = spans.Normalize()
		blk.SetSpans(spans)
	}

	m.clearHandleSelection()
	return m.markDirty()
}

// toggleFormatOnHandleSelectedLines toggles a format on all handle-selected lines
func (m *Model) toggleFormatOnHandleSelectedLines(toggleBold, toggleItalic, toggleUnderline bool) tea.Cmd {
	if !m.hasHandleSelection() {
		return nil
	}

	for idx := range m.selectedLines {
		blk := m.doc.BlockAt(idx)
		if blk == nil {
			continue
		}
		content := blk.Content()
		contentLen := len([]rune(content))
		if contentLen == 0 {
			continue
		}

		spans := blk.Spans()
		if toggleBold {
			spans = spans.ToggleBold(0, contentLen)
		}
		if toggleItalic {
			spans = spans.ToggleItalic(0, contentLen)
		}
		if toggleUnderline {
			spans = spans.ToggleUnderline(0, contentLen)
		}
		blk.SetSpans(spans)
	}

	m.clearHandleSelection()
	return m.markDirty()
}

// toggleFormatOnCharSelection toggles a format on the character selection
func (m *Model) toggleFormatOnCharSelection(toggleBold, toggleItalic, toggleUnderline bool) tea.Cmd {
	if !m.charSelect {
		return nil
	}

	// Normalize selection direction
	startLine, startCol := m.charSelStartLine, m.charSelStartCol
	endLine, endCol := m.charSelEndLine, m.charSelEndCol
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	// Apply format to each line in selection
	for line := startLine; line <= endLine && line < m.doc.BlockCount(); line++ {
		blk := m.doc.BlockAt(line)
		if blk == nil {
			continue
		}

		contentLen := len([]rune(blk.Content()))
		if contentLen == 0 {
			continue
		}

		// Determine start and end for this line
		lineStart := 0
		lineEnd := contentLen

		if line == startLine {
			lineStart = startCol
		}
		if line == endLine {
			lineEnd = endCol
		}

		if lineStart >= lineEnd {
			continue
		}

		spans := blk.Spans()
		if toggleBold {
			spans = spans.ToggleBold(lineStart, lineEnd)
		}
		if toggleItalic {
			spans = spans.ToggleItalic(lineStart, lineEnd)
		}
		if toggleUnderline {
			spans = spans.ToggleUnderline(lineStart, lineEnd)
		}
		blk.SetSpans(spans)
	}

	m.clearCharSelection()
	return m.markDirty()
}
