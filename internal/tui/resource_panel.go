package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/omnicli/omnicli/internal/mcp"
)

// ResourcePanel is a collapsible side panel that displays pinned MCP resources.
type ResourcePanel struct {
	resources []mcp.PinnedResource
	width     int
	height    int
	visible   bool
}

// NewResourcePanel creates a new empty resource panel.
func NewResourcePanel() ResourcePanel {
	return ResourcePanel{}
}

// SetResources updates the displayed resources.
func (p *ResourcePanel) SetResources(resources []mcp.PinnedResource) {
	p.resources = resources
}

// SetSize updates the panel dimensions.
func (p *ResourcePanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Toggle shows or hides the panel.
func (p *ResourcePanel) Toggle() {
	p.visible = !p.visible
}

// Visible returns whether the panel is currently shown.
func (p *ResourcePanel) Visible() bool {
	return p.visible
}

// Show makes the panel visible.
func (p *ResourcePanel) Show() {
	p.visible = true
}

// Hide makes the panel invisible.
func (p *ResourcePanel) Hide() {
	p.visible = false
}

// View renders the resource panel.
func (p ResourcePanel) View() string {
	if !p.visible || p.width <= 0 || p.height <= 0 {
		return ""
	}

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3")).
		Padding(0, 1)

	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Italic(true).
		Padding(0, 1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("240")).
		Width(p.width).
		Height(p.height)

	var b strings.Builder
	b.WriteString(headerStyle.Render("📎 Pinned Resources") + "\n")

	if len(p.resources) == 0 {
		b.WriteString(emptyStyle.Render("No resources pinned.\nUse /attach to add."))
	} else {
		for i, r := range p.resources {
			if i > 0 {
				b.WriteString("\n")
			}
			name := r.Name
			if name == "" {
				name = r.URI
			}
			line := fmt.Sprintf("%s", name)
			if len(line) > p.width-2 {
				line = line[:p.width-5] + "..."
			}
			b.WriteString(itemStyle.Render(line) + "\n")

			serverLine := fmt.Sprintf("  └ %s", r.ServerName)
			if len(serverLine) > p.width-2 {
				serverLine = serverLine[:p.width-5] + "..."
			}
			b.WriteString(descStyle.Render(serverLine) + "\n")

			if r.Description != "" {
				desc := r.Description
				if len(desc) > p.width-2 {
					desc = desc[:p.width-5] + "..."
				}
				b.WriteString(descStyle.Render("  "+desc) + "\n")
			}
		}
	}

	content := b.String()
	// Pad to fill height
	lines := strings.Split(content, "\n")
	for len(lines) < p.height {
		lines = append(lines, "")
	}
	if len(lines) > p.height {
		lines = lines[:p.height]
	}
	content = strings.Join(lines, "\n")

	return boxStyle.Render(content)
}
