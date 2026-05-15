package block

// TextBlock represents a plain text paragraph block
type TextBlock struct {
	BaseBlock
}

// NewTextBlock creates a new TextBlock with the given ID and content
func NewTextBlock(id, content string) *TextBlock {
	return &TextBlock{
		BaseBlock: NewBaseBlock(id, content),
	}
}

// Type returns the block type (TypeText)
func (b *TextBlock) Type() BlockType {
	return TypeText
}
