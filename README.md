# claudenelson

A terminal-based block editor written in Go, inspired by Notion. Features rich text formatting, multiple block types, and full compatibility with [Writeopia](https://writeopia.io) document format.

## Features

- **Block-based editing**: Organize content in discrete blocks like Notion
- **Rich text formatting**: Bold, Italic, Underline, and Highlight
- **Multiple block types**:
  - Text paragraphs
  - Headings (H1-H4)
  - Bullet lists
  - Checkboxes/Todo items
- **Writeopia compatible**: Documents save in Writeopia JSON format for cross-platform editing
- **Auto-save**: Changes are automatically saved after 500ms of inactivity
- **Block cursor**: Clean block-style cursor similar to modern terminal editors

## Installation

```bash
go install github.com/writeopiaproject/claudenelson@latest
```

Or build from source:

```bash
git clone https://github.com/writeopiaproject/claudenelson.git
cd claudenelson
go build -o claudenelson .
```

## Usage

```bash
# Open or create a document
claudenelson write --file mydocument.json

# Default file is document.json
claudenelson write
```

## Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `←` / `→` | Move cursor left/right |
| `↑` / `↓` | Move between blocks |
| `Home` / `Ctrl+A` | Move to start of line |
| `End` / `Ctrl+E` | Move to end of line |

### Editing
| Key | Action |
|-----|--------|
| `Enter` | Split block at cursor / Create new block |
| `Backspace` | Delete character / Merge with previous block |
| `Delete` | Delete character forward / Merge with next block |

### Formatting
| Key | Action |
|-----|--------|
| `Ctrl+B` | Toggle bold mode |
| `Ctrl+I` | Toggle italic mode |
| `Ctrl+U` | Toggle underline mode |
| `Ctrl+H` | Enter highlight selection mode |

### Multi-Line Selection (Whole Lines)
| Key | Action |
|-----|--------|
| `Option+↑` | Start/extend line selection upward |
| `Option+↓` | Start/extend line selection downward |
| `Backspace` / `Delete` | Delete all selected lines |
| `Esc` | Cancel selection |

Selected lines are highlighted with a purple background. The status bar shows "LINES:N" where N is the number of selected lines.

### Character Selection (Character by Character)
| Key | Action |
|-----|--------|
| `Option+←` | Start/extend selection leftward (can span lines) |
| `Option+→` | Start/extend selection rightward (can span lines) |
| `Backspace` / `Delete` | Delete selected characters |
| Any character | Replace selection with typed character |
| `Esc` | Cancel selection |

Selected characters are highlighted with a blue background. The status bar shows "SELECT" when character selection is active.

### Highlight Mode
When in highlight mode (`Ctrl+H`):
| Key | Action |
|-----|--------|
| `←` / `→` | Extend selection |
| `Home` / `End` | Select to start/end of line |
| `Enter` / `Space` / Any key | Apply highlight and exit |
| `Esc` | Cancel without applying |

Applying highlight to already-highlighted text will remove the highlight (toggle behavior).

### Block Type Triggers
Type these at the start of a line and press `Space`:
| Trigger | Creates |
|---------|---------|
| `#` | Heading 1 |
| `##` | Heading 2 |
| `###` | Heading 3 |
| `####` | Heading 4 |
| `-` | Bullet list item |
| `[]` | Unchecked checkbox |
| `[x]` | Checked checkbox |

### Undo/Redo
| Key | Action |
|-----|--------|
| `Ctrl+Z` | Undo last change |
| `Ctrl+Y` | Redo last undone change |

The undo system efficiently stores only the changed blocks (not the entire document) and supports up to 100 undo levels.

### Other
| Key | Action |
|-----|--------|
| `Ctrl+C` | Save and quit |
| Mouse click | Select block / Toggle checkbox |

## Document Format

Documents are saved in Writeopia-compatible JSON format (`.wrdoc.json`), allowing seamless editing between claudenelson and Writeopia apps.

Example document structure:
```json
{
    "id": "abc123xyz",
    "title": "My Document",
    "workspaceId": "disconnected_user",
    "content": [
        {
            "id": "block-1",
            "type": { "name": "title", "number": 11 },
            "text": "My Document",
            "position": 0
        },
        {
            "id": "block-2",
            "type": { "name": "message", "number": 0 },
            "text": "This is a paragraph with formatting.",
            "position": 1,
            "spans": [
                { "start": 0, "end": 4, "span": "BOLD" }
            ]
        }
    ],
    "createdAt": 1776609896651,
    "lastUpdatedAt": 1776609896651,
    "parentId": "root"
}
```

## Architecture

```
claudenelson/
├── cmd/                    # CLI commands (Cobra)
├── editor/
│   ├── block/              # Block types (text, heading, list, checkbox)
│   ├── document/           # Document model and operations
│   ├── drawer/             # Block rendering with formatting
│   ├── factory/            # Block creation
│   ├── format/             # Text formatting (spans, styles)
│   ├── persistence/        # JSON serialization (Writeopia format)
│   ├── styles/             # Terminal styles (lipgloss)
│   └── editor.go           # Main editor logic (bubbletea)
└── main.go
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [Cobra](https://github.com/spf13/cobra) - CLI framework

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License
