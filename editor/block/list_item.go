package block

// ListItemBlock represents a list item block
type ListItemBlock struct {
	BaseBlock
}

// NewListItemBlock creates a new ListItemBlock with the given ID and content
func NewListItemBlock(id, content string) *ListItemBlock {
	return &ListItemBlock{
		BaseBlock: NewBaseBlock(id, content),
	}
}

// Type returns the block type (TypeListItem)
func (b *ListItemBlock) Type() BlockType {
	return TypeListItem
}
