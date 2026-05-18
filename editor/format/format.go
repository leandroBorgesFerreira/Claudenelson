package format

// Style represents text formatting options
type Style struct {
	Bold      bool
	Italic    bool
	Underline bool
	Highlight bool
}

// IsPlain returns true if no formatting is applied
func (s Style) IsPlain() bool {
	return !s.Bold && !s.Italic && !s.Underline && !s.Highlight
}

// Span represents a range of text with formatting
type Span struct {
	Start int   // Start position (inclusive)
	End   int   // End position (exclusive)
	Style Style // Formatting for this range
}

// Spans is a list of formatting spans for a block
type Spans []Span

// Normalize merges adjacent spans with same formatting and removes empty spans
func (spans Spans) Normalize() Spans {
	if len(spans) == 0 {
		return spans
	}

	result := make(Spans, 0, len(spans))
	for _, span := range spans {
		// Skip empty spans
		if span.Start >= span.End {
			continue
		}
		// Skip plain spans (no formatting)
		if span.Style.IsPlain() {
			continue
		}

		// Try to merge with previous span
		if len(result) > 0 {
			prev := &result[len(result)-1]
			if prev.End == span.Start && prev.Style == span.Style {
				prev.End = span.End
				continue
			}
		}
		result = append(result, span)
	}
	return result
}

// InsertAt adjusts spans when a character is inserted at position pos
// If style is not plain, creates or extends a span at that position
func (spans Spans) InsertAt(pos int, style Style) Spans {
	result := make(Spans, 0, len(spans)+1)

	for _, span := range spans {
		if span.End <= pos {
			// Span is entirely before insertion point
			result = append(result, span)
		} else if span.Start >= pos {
			// Span is entirely after insertion point - shift it
			result = append(result, Span{
				Start: span.Start + 1,
				End:   span.End + 1,
				Style: span.Style,
			})
		} else {
			// Insertion point is inside the span - split it
			if span.Start < pos {
				result = append(result, Span{
					Start: span.Start,
					End:   pos,
					Style: span.Style,
				})
			}
			// The character being inserted will be handled below
			if pos < span.End {
				result = append(result, Span{
					Start: pos + 1,
					End:   span.End + 1,
					Style: span.Style,
				})
			}
		}
	}

	// Add span for the new character if it has formatting
	if !style.IsPlain() {
		result = append(result, Span{
			Start: pos,
			End:   pos + 1,
			Style: style,
		})
	}

	return result.Normalize()
}

// DeleteAt adjusts spans when a character is deleted at position pos
func (spans Spans) DeleteAt(pos int) Spans {
	result := make(Spans, 0, len(spans))

	for _, span := range spans {
		if span.End <= pos {
			// Span is entirely before deletion point
			result = append(result, span)
		} else if span.Start > pos {
			// Span is entirely after deletion point - shift it back
			result = append(result, Span{
				Start: span.Start - 1,
				End:   span.End - 1,
				Style: span.Style,
			})
		} else {
			// Deletion point is inside or at the edge of the span
			newSpan := Span{
				Start: span.Start,
				End:   span.End - 1,
				Style: span.Style,
			}
			if newSpan.Start < newSpan.End {
				result = append(result, newSpan)
			}
		}
	}

	return result.Normalize()
}

// StyleAt returns the combined style at a given position
func (spans Spans) StyleAt(pos int) Style {
	var style Style
	for _, span := range spans {
		if span.Start <= pos && pos < span.End {
			// Combine styles (multiple spans can overlap conceptually after edits)
			if span.Style.Bold {
				style.Bold = true
			}
			if span.Style.Italic {
				style.Italic = true
			}
			if span.Style.Underline {
				style.Underline = true
			}
			if span.Style.Highlight {
				style.Highlight = true
			}
		}
	}
	return style
}

// isRangeHighlighted checks if the entire range [start, end) is highlighted
func (spans Spans) isRangeHighlighted(start, end int) bool {
	if start >= end {
		return false
	}
	// Check each position in the range
	for pos := start; pos < end; pos++ {
		highlighted := false
		for _, span := range spans {
			if span.Style.Highlight && span.Start <= pos && pos < span.End {
				highlighted = true
				break
			}
		}
		if !highlighted {
			return false
		}
	}
	return true
}

// ToggleHighlight toggles highlight formatting for a range of text
// If the entire range is highlighted, removes the highlight; otherwise adds it
func (spans Spans) ToggleHighlight(start, end int) Spans {
	if start >= end {
		return spans
	}

	if spans.isRangeHighlighted(start, end) {
		// Remove highlight from this range
		return spans.RemoveHighlight(start, end)
	}

	// Add highlight to this range
	result := append(spans, Span{
		Start: start,
		End:   end,
		Style: Style{Highlight: true},
	})
	return result.Normalize()
}

// RemoveHighlight removes highlight formatting from a range of text
func (spans Spans) RemoveHighlight(start, end int) Spans {
	if start >= end {
		return spans
	}

	var result Spans
	for _, span := range spans {
		if !span.Style.Highlight {
			// Keep non-highlight spans as-is
			result = append(result, span)
			continue
		}

		// Handle highlight spans - may need to split or remove
		if span.End <= start || span.Start >= end {
			// Span is completely outside the range - keep it
			result = append(result, span)
		} else if span.Start >= start && span.End <= end {
			// Span is completely inside the range - remove it (don't add)
		} else if span.Start < start && span.End > end {
			// Range is inside the span - split into two parts
			result = append(result, Span{
				Start: span.Start,
				End:   start,
				Style: span.Style,
			})
			result = append(result, Span{
				Start: end,
				End:   span.End,
				Style: span.Style,
			})
		} else if span.Start < start {
			// Span overlaps on the left - keep left part
			result = append(result, Span{
				Start: span.Start,
				End:   start,
				Style: span.Style,
			})
		} else if span.End > end {
			// Span overlaps on the right - keep right part
			result = append(result, Span{
				Start: end,
				End:   span.End,
				Style: span.Style,
			})
		}
	}

	return result.Normalize()
}

// isRangeBold checks if the entire range [start, end) is bold
func (spans Spans) isRangeBold(start, end int) bool {
	if start >= end {
		return false
	}
	for pos := start; pos < end; pos++ {
		bold := false
		for _, span := range spans {
			if span.Style.Bold && span.Start <= pos && pos < span.End {
				bold = true
				break
			}
		}
		if !bold {
			return false
		}
	}
	return true
}

// isRangeItalic checks if the entire range [start, end) is italic
func (spans Spans) isRangeItalic(start, end int) bool {
	if start >= end {
		return false
	}
	for pos := start; pos < end; pos++ {
		italic := false
		for _, span := range spans {
			if span.Style.Italic && span.Start <= pos && pos < span.End {
				italic = true
				break
			}
		}
		if !italic {
			return false
		}
	}
	return true
}

// isRangeUnderline checks if the entire range [start, end) is underlined
func (spans Spans) isRangeUnderline(start, end int) bool {
	if start >= end {
		return false
	}
	for pos := start; pos < end; pos++ {
		underline := false
		for _, span := range spans {
			if span.Style.Underline && span.Start <= pos && pos < span.End {
				underline = true
				break
			}
		}
		if !underline {
			return false
		}
	}
	return true
}

// ToggleBold toggles bold formatting for a range of text
func (spans Spans) ToggleBold(start, end int) Spans {
	if start >= end {
		return spans
	}

	if spans.isRangeBold(start, end) {
		return spans.RemoveFormat(start, end, true, false, false, false)
	}

	result := append(spans, Span{
		Start: start,
		End:   end,
		Style: Style{Bold: true},
	})
	return result.Normalize()
}

// ToggleItalic toggles italic formatting for a range of text
func (spans Spans) ToggleItalic(start, end int) Spans {
	if start >= end {
		return spans
	}

	if spans.isRangeItalic(start, end) {
		return spans.RemoveFormat(start, end, false, true, false, false)
	}

	result := append(spans, Span{
		Start: start,
		End:   end,
		Style: Style{Italic: true},
	})
	return result.Normalize()
}

// ToggleUnderline toggles underline formatting for a range of text
func (spans Spans) ToggleUnderline(start, end int) Spans {
	if start >= end {
		return spans
	}

	if spans.isRangeUnderline(start, end) {
		return spans.RemoveFormat(start, end, false, false, true, false)
	}

	result := append(spans, Span{
		Start: start,
		End:   end,
		Style: Style{Underline: true},
	})
	return result.Normalize()
}

// RemoveFormat removes specific formatting from a range of text
func (spans Spans) RemoveFormat(start, end int, removeBold, removeItalic, removeUnderline, removeHighlight bool) Spans {
	if start >= end {
		return spans
	}

	var result Spans
	for _, span := range spans {
		// Check if this span has any of the formats we're removing
		shouldProcess := (removeBold && span.Style.Bold) ||
			(removeItalic && span.Style.Italic) ||
			(removeUnderline && span.Style.Underline) ||
			(removeHighlight && span.Style.Highlight)

		if !shouldProcess {
			result = append(result, span)
			continue
		}

		// Handle spans that need format removal
		if span.End <= start || span.Start >= end {
			// Span is completely outside the range - keep it
			result = append(result, span)
		} else if span.Start >= start && span.End <= end {
			// Span is completely inside the range - remove the specified formats
			newStyle := span.Style
			if removeBold {
				newStyle.Bold = false
			}
			if removeItalic {
				newStyle.Italic = false
			}
			if removeUnderline {
				newStyle.Underline = false
			}
			if removeHighlight {
				newStyle.Highlight = false
			}
			if !newStyle.IsPlain() {
				result = append(result, Span{Start: span.Start, End: span.End, Style: newStyle})
			}
		} else if span.Start < start && span.End > end {
			// Range is inside the span - split into three parts
			result = append(result, Span{Start: span.Start, End: start, Style: span.Style})
			newStyle := span.Style
			if removeBold {
				newStyle.Bold = false
			}
			if removeItalic {
				newStyle.Italic = false
			}
			if removeUnderline {
				newStyle.Underline = false
			}
			if removeHighlight {
				newStyle.Highlight = false
			}
			if !newStyle.IsPlain() {
				result = append(result, Span{Start: start, End: end, Style: newStyle})
			}
			result = append(result, Span{Start: end, End: span.End, Style: span.Style})
		} else if span.Start < start {
			// Span overlaps on the left
			result = append(result, Span{Start: span.Start, End: start, Style: span.Style})
			newStyle := span.Style
			if removeBold {
				newStyle.Bold = false
			}
			if removeItalic {
				newStyle.Italic = false
			}
			if removeUnderline {
				newStyle.Underline = false
			}
			if removeHighlight {
				newStyle.Highlight = false
			}
			if !newStyle.IsPlain() {
				result = append(result, Span{Start: start, End: span.End, Style: newStyle})
			}
		} else if span.End > end {
			// Span overlaps on the right
			newStyle := span.Style
			if removeBold {
				newStyle.Bold = false
			}
			if removeItalic {
				newStyle.Italic = false
			}
			if removeUnderline {
				newStyle.Underline = false
			}
			if removeHighlight {
				newStyle.Highlight = false
			}
			if !newStyle.IsPlain() {
				result = append(result, Span{Start: span.Start, End: end, Style: newStyle})
			}
			result = append(result, Span{Start: end, End: span.End, Style: span.Style})
		}
	}

	return result.Normalize()
}

// SplitAt splits spans at position and returns spans for left and right parts
func (spans Spans) SplitAt(pos int) (Spans, Spans) {
	var left, right Spans

	for _, span := range spans {
		if span.End <= pos {
			// Span is entirely in left part
			left = append(left, span)
		} else if span.Start >= pos {
			// Span is entirely in right part - adjust positions
			right = append(right, Span{
				Start: span.Start - pos,
				End:   span.End - pos,
				Style: span.Style,
			})
		} else {
			// Span crosses the split point
			left = append(left, Span{
				Start: span.Start,
				End:   pos,
				Style: span.Style,
			})
			right = append(right, Span{
				Start: 0,
				End:   span.End - pos,
				Style: span.Style,
			})
		}
	}

	return left.Normalize(), right.Normalize()
}

// Append adds spans from another block, adjusting positions by offset
func (spans Spans) Append(other Spans, offset int) Spans {
	result := make(Spans, len(spans), len(spans)+len(other))
	copy(result, spans)

	for _, span := range other {
		result = append(result, Span{
			Start: span.Start + offset,
			End:   span.End + offset,
			Style: span.Style,
		})
	}

	return result.Normalize()
}
