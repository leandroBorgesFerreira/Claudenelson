package block

// BlockType represents the type of a block
type BlockType int

const (
	TypeText BlockType = iota
	TypeListItem
	TypeCheckboxItem
	TypeH1
	TypeH2
	TypeH3
	TypeH4
)

// String returns the string representation of a BlockType
func (bt BlockType) String() string {
	switch bt {
	case TypeText:
		return "text"
	case TypeListItem:
		return "list_item"
	case TypeCheckboxItem:
		return "checkbox"
	case TypeH1:
		return "h1"
	case TypeH2:
		return "h2"
	case TypeH3:
		return "h3"
	case TypeH4:
		return "h4"
	default:
		return "unknown"
	}
}

// Block is the interface that all block types must implement
type Block interface {
	ID() string
	Type() BlockType
	Content() string
	SetContent(content string)
	Metadata() map[string]interface{}
	SetMetadata(key string, value interface{})
}

// BaseBlock provides common functionality for all block types
type BaseBlock struct {
	id       string
	content  string
	metadata map[string]interface{}
}

// NewBaseBlock creates a new BaseBlock with the given ID and content
func NewBaseBlock(id, content string) BaseBlock {
	return BaseBlock{
		id:       id,
		content:  content,
		metadata: make(map[string]interface{}),
	}
}

// ID returns the unique identifier of the block
func (b *BaseBlock) ID() string {
	return b.id
}

// Content returns the text content of the block
func (b *BaseBlock) Content() string {
	return b.content
}

// SetContent sets the text content of the block
func (b *BaseBlock) SetContent(content string) {
	b.content = content
}

// Metadata returns the metadata map of the block
func (b *BaseBlock) Metadata() map[string]interface{} {
	return b.metadata
}

// SetMetadata sets a metadata key-value pair
func (b *BaseBlock) SetMetadata(key string, value interface{}) {
	b.metadata[key] = value
}
