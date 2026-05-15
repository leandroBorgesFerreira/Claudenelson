package block

// HeadingBlock represents a heading block (H1-H4)
type HeadingBlock struct {
	BaseBlock
	level int // 1-4 for H1-H4
}

// NewHeadingBlock creates a new HeadingBlock with the given ID, content, and level
func NewHeadingBlock(id, content string, level int) *HeadingBlock {
	if level < 1 {
		level = 1
	}
	if level > 4 {
		level = 4
	}
	return &HeadingBlock{
		BaseBlock: NewBaseBlock(id, content),
		level:     level,
	}
}

// Type returns the block type based on the heading level
func (b *HeadingBlock) Type() BlockType {
	switch b.level {
	case 1:
		return TypeH1
	case 2:
		return TypeH2
	case 3:
		return TypeH3
	case 4:
		return TypeH4
	default:
		return TypeH1
	}
}

// Level returns the heading level (1-4)
func (b *HeadingBlock) Level() int {
	return b.level
}
