package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ResourceBrowser is a Bubble Tea sub-model for selecting an MCP resource.
type ResourceBrowser struct {
	items    []ResourceItem
	cursor   int
	width    int
	height   int
	selected bool
}

// NewResourceBrowser creates a new resource browser with the given dimensions.
func NewResourceBrowser(width, height int) ResourceBrowser {
	return ResourceBrowser{
		width:  width,
		height: height,
	}
}

// SetItems updates the list of available resources.
func (b *ResourceBrowser) SetItems(items []ResourceItem) {
	b.items = items
	b.cursor = 0
	b.selected = false
}

// SetSize updates the browser dimensions.
func (b *ResourceBrowser) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// MoveDown moves the cursor down.
func (b *ResourceBrowser) MoveDown() {
	if b.cursor < len(b.items)-1 {
		b.cursor++
	}
}

// MoveUp moves the cursor up.
func (b *ResourceBrowser) MoveUp() {
	if b.cursor > 0 {
		b.cursor--
	}
}

// Selected returns the currently selected item, or nil if none.
func (b *ResourceBrowser) Selected() *ResourceItem {
	if b.selected && b.cursor >= 0 && b.cursor < len(b.items) {
		item := b.items[b.cursor]
		return &item
	}
	return nil
}

// MarkSelected marks the current item as selected.
func (b *ResourceBrowser) MarkSelected() {
	if b.cursor >= 0 && b.cursor < len(b.items) {
		b.selected = true
	}
}

// Update handles keyboard navigation.
func (b ResourceBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			b.MoveDown()
		case "k", "up":
			b.MoveUp()
		}
	}
	return b, nil
}

// Init returns the initial command.
func (b ResourceBrowser) Init() tea.Cmd {
	return nil
}

// View renders the resource browser as a centered overlay.
func (b ResourceBrowser) View() string {
	boxWidth := b.width - 8
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

	serverStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)

	var body strings.Builder
	if len(b.items) == 0 {
		body.WriteString("Loading resources...\n")
	} else {
		for i, item := range b.items {
			name := item.Name
			if name == "" {
				name = item.URI
			}

			line := fmt.Sprintf(" %s  %s", mark(i == b.cursor), truncate(name, boxWidth-6))
			if i == b.cursor {
				body.WriteString(selectedStyle.Render(line))
			} else {
				body.WriteString(itemStyle.Render(line))
			}
			body.WriteString("\n")

			if item.Description != "" {
				desc := truncate(item.Description, boxWidth-8)
				body.WriteString(serverStyle.Render("     " + desc))
				body.WriteString("\n")
			}

			serverLine := fmt.Sprintf("     [%s]", item.ServerName)
			body.WriteString(serverStyle.Render(serverLine))
			body.WriteString("\n")
		}
	}

	var bld strings.Builder
	bld.WriteString(titleStyle.Render("📎 Attach MCP Resource") + "\n")
	bld.WriteString(body.String())
	bld.WriteString(hintStyle.Render("[↑/↓] navigate  [Enter] select  [Esc] cancel") + "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(bld.String())

	return lipgloss.Place(b.width, b.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

func mark(selected bool) string {
	if selected {
		return ">"
	}
	return " "
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
