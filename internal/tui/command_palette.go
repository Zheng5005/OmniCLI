package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type paletteItem struct {
	name        string
	description string
}

// CommandPalette is a Bubble Tea sub-model for selecting slash commands.
type CommandPalette struct {
	input    textinput.Model
	items    []paletteItem
	filtered []paletteItem
	cursor   int
	width    int
	height   int
}

// NewCommandPalette creates a new command palette with the given commands.
func NewCommandPalette(commands []CommandDesc) *CommandPalette {
	items := make([]paletteItem, len(commands))
	for i, cmd := range commands {
		items[i] = paletteItem{name: cmd.Name, description: cmd.Description}
	}

	ti := textinput.New()
	ti.Placeholder = "Filter commands..."
	_ = ti.Focus()

	return &CommandPalette{
		input:    ti,
		items:    items,
		filtered: items,
		cursor:   0,
	}
}

// SetSize updates the palette dimensions.
func (p *CommandPalette) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Init returns the initial command.
func (p CommandPalette) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles keyboard input for the palette.
func (p CommandPalette) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			return p, func() tea.Msg {
				return CommandPaletteDismissMsg{}
			}
		case tea.KeyEnter:
			if len(p.filtered) > 0 && p.cursor >= 0 && p.cursor < len(p.filtered) {
				selected := p.filtered[p.cursor]
				return p, func() tea.Msg {
					return CommandPaletteExecuteMsg{Command: selected.name}
				}
			}
			return p, nil
		case tea.KeyUp:
			p.moveUp()
			return p, nil
		case tea.KeyDown:
			p.moveDown()
			return p, nil
		}

		switch msg.String() {
		case "k":
			p.moveUp()
			return p, nil
		case "j":
			p.moveDown()
			return p, nil
		}

		// Let textinput handle other keys (typing).
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		p.applyFilter()
		return p, cmd
	}

	return p, nil
}

func (p *CommandPalette) moveUp() {
	if len(p.filtered) == 0 {
		return
	}
	p.cursor--
	if p.cursor < 0 {
		p.cursor = len(p.filtered) - 1
	}
}

func (p *CommandPalette) moveDown() {
	if len(p.filtered) == 0 {
		return
	}
	p.cursor++
	if p.cursor >= len(p.filtered) {
		p.cursor = 0
	}
}

func (p *CommandPalette) applyFilter() {
	filter := strings.ToLower(p.input.Value())
	if filter == "" {
		p.filtered = p.items
		p.cursor = 0
		return
	}

	p.filtered = p.filtered[:0]
	for _, item := range p.items {
		if strings.HasPrefix(strings.ToLower(item.name), filter) {
			p.filtered = append(p.filtered, item)
		}
	}
	p.cursor = 0
}

// View renders the command palette as a centered overlay.
func (p CommandPalette) View() string {
	boxWidth := p.width - 8
	if boxWidth > 80 {
		boxWidth = 80
	}
	if boxWidth < 40 {
		boxWidth = 40
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3")).
		MarginBottom(1)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		MarginTop(1)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("6")).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)

	inputStyle := lipgloss.NewStyle().
		MarginBottom(1)

	var body strings.Builder

	body.WriteString(titleStyle.Render("⌘ Command Palette") + "\n")

	// Filter input
	body.WriteString(inputStyle.Render(p.input.View()))
	body.WriteString("\n")

	// Items
	if len(p.filtered) == 0 {
		body.WriteString("No matching commands\n")
	} else {
		start, end := p.visibleRange()
		for i := start; i < end; i++ {
			item := p.filtered[i]
			name := truncate(item.name, boxWidth-4)
			desc := truncate(item.description, boxWidth-6)

			line := " " + mark(i == p.cursor) + " " + name
			if i == p.cursor {
				body.WriteString(selectedStyle.Render(line))
			} else {
				body.WriteString(itemStyle.Render(line))
			}
			body.WriteString("\n")

			if desc != "" {
				body.WriteString(descStyle.Render("    " + desc))
				body.WriteString("\n")
			}
		}
	}

	body.WriteString(hintStyle.Render("[↑/↓] navigate  [Enter] select  [Esc] cancel") + "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(body.String())

	return lipgloss.Place(p.width, p.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

func (p CommandPalette) visibleRange() (start, end int) {
	total := len(p.filtered)
	if total == 0 {
		return 0, 0
	}
	maxVisible := 15
	if total <= maxVisible {
		return 0, total
	}
	start = p.cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}
	return start, end
}
