# Claudenelson Manual for Claude Code

This manual provides context for Claude Code to understand and modify the claudenelson terminal text editor. Claudenelson is a Notion-like block-based editor written in Go using BubbleTea.

## Project Overview

Claudenelson is a terminal-based rich text editor with:
- Block-based document structure (like Notion)
- Rich text formatting (bold, italic, underline, highlight)
- Multiple block types (text, headings, lists, checkboxes)
- Writeopia-compatible JSON document format
- Undo/redo with efficient delta storage
- Multi-line and character selection modes

---

## Architecture

### Directory Structure

```
claudenelson/
├── main.go                 # Entry point
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command
│   ├── write.go           # `write` command - launches editor
│   └── read.go            # `read` command - prints document
├── editor/
│   ├── editor.go          # Main editor logic, key handling, UI
│   ├── block/             # Block type definitions
│   │   ├── block.go       # Block interface and types
│   │   ├── text.go        # TextBlock
│   │   ├── heading.go     # HeadingBlock (H1-H4)
│   │   ├── list_item.go   # ListItemBlock
│   │   └── checkbox.go    # CheckboxBlock
│   ├── document/          # Document model
│   │   └── document.go    # Document struct, cursor, block operations
│   ├── drawer/            # Block rendering
│   │   ├── drawer.go      # DrawContext, rendering functions
│   │   ├── text_drawer.go
│   │   ├── heading_drawer.go
│   │   ├── list_drawer.go
│   │   └── checkbox_drawer.go
│   ├── factory/           # Block creation
│   │   └── factory.go     # BlockFactory
│   ├── format/            # Text formatting
│   │   └── format.go      # Style, Span, Spans operations
│   ├── persistence/       # File I/O
│   │   └── persistence.go # Save/Load, Writeopia format
│   ├── styles/            # Terminal styles
│   │   └── styles.go      # Lipgloss styles
│   └── undo/              # Undo/redo system
│       └── undo.go        # Operation types, Stack, Manager
└── .github/workflows/     # CI/CD
    └── ci.yml
```

### Key Dependencies

- `github.com/charmbracelet/bubbletea` - Terminal UI framework (Elm architecture)
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/spf13/cobra` - CLI framework

---

## Block System

### Block Interface

All blocks implement the `Block` interface in `editor/block/block.go`:

```go
type Block interface {
    ID() string
    Type() BlockType
    Content() string
    SetContent(string)
    Spans() format.Spans
    SetSpans(format.Spans)
}
```

### Block Types

| Type | Constant | Description |
|------|----------|-------------|
| Text | `TypeText` | Plain text paragraph |
| H1 | `TypeH1` | Heading level 1 |
| H2 | `TypeH2` | Heading level 2 |
| H3 | `TypeH3` | Heading level 3 |
| H4 | `TypeH4` | Heading level 4 |
| List Item | `TypeListItem` | Bullet point (•) |
| Checkbox | `TypeCheckboxItem` | Todo item with checked state |

### Block Creation

Use `BlockFactory` in `editor/factory/factory.go`:

```go
f := factory.NewBlockFactory()
f.CreateText("content")
f.CreateHeading("title", 1)  // level 1-4
f.CreateListItem("item")
f.CreateCheckbox("todo", false)  // checked state
f.CreateTextWithSpans("formatted", spans)
f.CreateFromLine("# Heading")  // Parses markdown-like syntax
```

---

## Formatting System

### Style Structure

In `editor/format/format.go`:

```go
type Style struct {
    Bold      bool
    Italic    bool
    Underline bool
    Highlight bool
}
```

### Span Structure

```go
type Span struct {
    Start int    // Start index (inclusive)
    End   int    // End index (exclusive)
    Style Style
}

type Spans []Span
```

### Spans Operations

| Method | Description |
|--------|-------------|
| `spans.InsertAt(pos)` | Adjust spans for character insertion |
| `spans.DeleteAt(pos)` | Adjust spans for character deletion |
| `spans.SplitAt(pos)` | Split spans at position (for Enter key) |
| `spans.StyleAt(pos)` | Get combined style at position |
| `spans.ToggleHighlight(start, end)` | Add/remove highlight |
| `spans.Normalize()` | Merge adjacent spans with same style |

---

## Editor State

### Model Structure

In `editor/editor.go`:

```go
type Model struct {
    doc       *document.Document
    factory   *factory.BlockFactory
    registry  *drawer.DrawerRegistry
    width     int
    height    int
    savePath  string
    dirty     bool
    saveTimer time.Time

    // Formatting modes
    boldMode      bool
    italicMode    bool
    underlineMode bool

    // Highlight mode (within single line)
    highlightMode  bool
    selectionStart int

    // Multi-line selection (whole lines)
    multiLineSelect    bool
    lineSelectionStart int
    lineSelectionEnd   int

    // Character selection (spans lines)
    charSelect       bool
    charSelStartLine int
    charSelStartCol  int
    charSelEndLine   int
    charSelEndCol    int

    // Undo/Redo
    undoManager       *undo.Manager
    lastBlockState    *undo.BlockState
    lastBlockIndex    int
    pendingUndoRecord bool
}
```

### Document Structure

In `editor/document/document.go`:

```go
type Document struct {
    Blocks     []block.Block
    CursorLine int
    CursorCol  int
}
```

---

## Selection Modes

### 1. Highlight Mode (`Ctrl+H`)

For applying yellow highlight formatting within a single line.

- Enter: `Ctrl+H`
- Extend: `←/→` arrow keys
- Apply: `Enter`, `Space`, or any character
- Cancel: `Esc`
- Toggle: Re-highlighting removes existing highlight

### 2. Line Selection Mode (`Option+↑/↓`)

For selecting whole lines.

- Enter: `Option+↑` or `Option+↓`
- Extend: Continue with `Option+↑/↓`
- Delete: `Backspace`, `Delete`, or `Enter`
- Cancel: `Esc` or any regular navigation

### 3. Character Selection Mode (`Option+←/→`)

For character-by-character selection that spans lines.

- Enter: `Option+←` or `Option+→`
- Extend: Continue with `Option+←/→`
- Delete: `Backspace` or `Delete`
- Replace: Type any character
- Cancel: `Esc` or any regular navigation

---

## Undo/Redo System

### Operation Types

In `editor/undo/undo.go`:

```go
const (
    OpModify      // Block content changed
    OpAdd         // Block added
    OpDelete      // Block deleted
    OpMultiDelete // Multiple blocks deleted
)
```

### BlockState

Captures minimal state for restoration:

```go
type BlockState struct {
    Content string
    Spans   format.Spans
    Checked *bool           // For checkboxes
    Type    block.BlockType
}
```

### Recording Operations

```go
m.undoManager.RecordModify(index, oldState, newState, cursorLine, cursorCol)
m.undoManager.RecordAdd(index, newState, cursorLine, cursorCol)
m.undoManager.RecordDelete(index, oldState, cursorLine, cursorCol)
m.undoManager.RecordMultiDelete(startIndex, []BlockState, cursorLine, cursorCol)
```

### Performing Undo/Redo

```go
op, ok := m.undoManager.Undo()  // Returns operation and moves to redo stack
op, ok := m.undoManager.Redo()  // Returns operation and moves to undo stack
```

---

## Key Bindings

### Navigation
| Key | Action |
|-----|--------|
| `←/→` | Move cursor left/right |
| `↑/↓` | Move between blocks |
| `Home/Ctrl+A` | Move to line start |
| `End/Ctrl+E` | Move to line end |

### Editing
| Key | Action |
|-----|--------|
| `Enter` | Split block at cursor |
| `Backspace` | Delete backward / merge blocks |
| `Delete` | Delete forward / merge blocks |

### Formatting
| Key | Action |
|-----|--------|
| `Ctrl+B` | Toggle bold mode |
| `Ctrl+I` | Toggle italic mode |
| `Ctrl+U` | Toggle underline mode |
| `Ctrl+H` | Enter highlight mode |

### Selection
| Key | Action |
|-----|--------|
| `Option+↑/↓` | Line selection |
| `Option+←/→` | Character selection |
| `Esc` | Cancel selection |

### Undo/Redo
| Key | Action |
|-----|--------|
| `Ctrl+Z` | Undo |
| `Ctrl+Y` | Redo |

### Block Triggers
Type at start of line + Space:
| Trigger | Creates |
|---------|---------|
| `#` | H1 |
| `##` | H2 |
| `###` | H3 |
| `####` | H4 |
| `-` | List item |
| `[]` | Unchecked checkbox |
| `[x]` | Checked checkbox |

---

## Persistence / File Format

### Writeopia Compatibility

Documents are saved in Writeopia-compatible JSON format for cross-platform editing.

### Document JSON Structure

```json
{
    "id": "unique_10_char_id",
    "title": "Document Title",
    "workspaceId": "disconnected_user",
    "content": [
        {
            "id": "block_id",
            "type": { "name": "title", "number": 11 },
            "text": "Document Title",
            "position": 0
        },
        {
            "id": "block_id",
            "type": { "name": "message", "number": 0 },
            "text": "Paragraph text",
            "position": 1,
            "spans": [
                { "start": 0, "end": 9, "span": "BOLD" }
            ],
            "tags": [
                { "tag": "H2", "position": 0 }
            ]
        }
    ],
    "createdAt": 1776609896651,
    "lastUpdatedAt": 1776609896651,
    "parentId": "root"
}
```

### Type Mapping

| Claudenelson | Writeopia Type | Number |
|--------------|----------------|--------|
| First block | `title` | 11 |
| Text | `message` | 0 |
| H1-H4 | `message` + tag | 0 |
| List Item | `unordered_list_item` | 16 |
| Checkbox | `check_item` | 10 |

### Span Types

| Format | Writeopia Span |
|--------|---------------|
| Bold | `BOLD` |
| Italic | `ITALIC` |
| Underline | `UNDERLINE` |
| Highlight | `HIGHLIGHT` |

---

## Rendering System

### DrawContext

Passed to all drawers:

```go
type DrawContext struct {
    Width          int
    IsFocused      bool
    CursorPos      int
    LineNumber     int
    ShowCursor     bool
    SelectionStart int   // -1 if no selection
    SelectionEnd   int   // -1 if no selection
    LineSelected   bool  // Multi-line selection
}
```

### Drawer Interface

```go
type Drawer interface {
    Draw(b block.Block, ctx DrawContext) string
    SupportedType() block.BlockType
}
```

### Rendering Functions

```go
RenderFormattedContent(content, spans, baseStyle)
RenderFormattedContentWithSelection(content, spans, baseStyle, selStart, selEnd)
RenderFormattedContentWithCursor(content, spans, baseStyle, cursorPos)
RenderFormattedContentFull(content, spans, baseStyle, cursorPos, selStart, selEnd, lineSelected)
```

### Cursor Style

Block cursor using `Reverse(true)` - character at cursor has inverted colors.

---

## Adding New Features

### Adding a New Block Type

1. Add constant in `editor/block/block.go`
2. Create block struct implementing `Block` interface
3. Add creation method to `editor/factory/factory.go`
4. Create drawer in `editor/drawer/`
5. Register drawer in `DrawerRegistry.RegisterAll()`
6. Update persistence for Writeopia mapping

### Adding a New Key Binding

1. Add handler in `editor/editor.go` `Update()` method
2. Update help text in `View()` method
3. Update README.md

### Adding a New Formatting Style

1. Add field to `Style` struct in `editor/format/format.go`
2. Update `IsPlain()` and `StyleAt()` methods
3. Add style in `editor/styles/styles.go`
4. Update rendering in `editor/drawer/drawer.go`
5. Update persistence for Writeopia span mapping

---

## CLI Commands

### write

Launches the editor:
```bash
claudenelson write --file document.json
claudenelson write -f document.json
```

### read

Prints document to terminal:
```bash
claudenelson read --file document.json
claudenelson read -f document.json
```

---

## Testing

Run tests:
```bash
go test ./...
```

Build:
```bash
go build ./...
```

---

## Auto-Save

- Debounced saves with 500ms delay
- Saves on `Ctrl+C` quit
- Atomic writes using temp file + rename
