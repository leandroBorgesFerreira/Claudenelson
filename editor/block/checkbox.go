package block

// CheckboxBlock represents a checkbox/todo item block
type CheckboxBlock struct {
	BaseBlock
	checked bool
}

// NewCheckboxBlock creates a new CheckboxBlock with the given ID, content, and checked state
func NewCheckboxBlock(id, content string, checked bool) *CheckboxBlock {
	return &CheckboxBlock{
		BaseBlock: NewBaseBlock(id, content),
		checked:   checked,
	}
}

// Type returns the block type (TypeCheckboxItem)
func (b *CheckboxBlock) Type() BlockType {
	return TypeCheckboxItem
}

// IsChecked returns whether the checkbox is checked
func (b *CheckboxBlock) IsChecked() bool {
	return b.checked
}

// Toggle toggles the checked state of the checkbox
func (b *CheckboxBlock) Toggle() {
	b.checked = !b.checked
}

// SetChecked sets the checked state of the checkbox
func (b *CheckboxBlock) SetChecked(checked bool) {
	b.checked = checked
}
