package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

var systemStyle = lipgloss.NewStyle().Italic(true).Faint(true)

// ViewportModel manages the scrollable chat display with markdown rendering.
type ViewportModel struct {
	viewport  viewport.Model
	content   string
	streaming string
	width     int
	height    int
	renderer  *glamour.TermRenderer
}

// NewViewportModel creates a new ViewportModel with the given dimensions.
func NewViewportModel(width, height int) ViewportModel {
	vp := viewport.New(width, height)
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.DarkStyleConfig),
		glamour.WithWordWrap(width),
	)

	return ViewportModel{
		viewport: vp,
		width:    width,
		height:   height,
		renderer: r,
	}
}

// Update delegates scroll handling to the underlying viewport.
func (m ViewportModel) Update(msg tea.Msg) (ViewportModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the viewport.
func (m ViewportModel) View() string {
	return m.viewport.View()
}

// AppendChunk appends a streaming chunk. During streaming the raw text is
// displayed for performance; glamour rendering happens in FinalizeResponse.
func (m *ViewportModel) AppendChunk(chunk string) {
	m.streaming += chunk
	m.viewport.SetContent(m.content + m.streaming)
	m.viewport.GotoBottom()
}

// FinalizeResponse renders the accumulated streaming buffer through glamour,
// appends it to the permanent content, and clears the streaming buffer.
func (m *ViewportModel) FinalizeResponse() {
	rendered, err := m.renderer.Render(m.streaming)
	if err != nil {
		rendered = m.streaming
	}
	m.content += strings.TrimRight(rendered, "\n") + "\n"
	m.streaming = ""
	m.viewport.SetContent(m.content)
	m.viewport.GotoBottom()
}

// AppendSystem appends a styled system message to the viewport.
func (m *ViewportModel) AppendSystem(text string) {
	m.content += systemStyle.Render(text) + "\n"
	m.viewport.SetContent(m.content + m.streaming)
	m.viewport.GotoBottom()
}

// AppendMarkdown renders markdown content and appends it to the viewport.
func (m *ViewportModel) AppendMarkdown(text string) {
	rendered, err := m.renderer.Render(text)
	if err != nil {
		rendered = text
	}
	m.content += strings.TrimRight(rendered, "\n") + "\n"
	m.viewport.SetContent(m.content + m.streaming)
	m.viewport.GotoBottom()
}

// SetSize resizes the viewport and recreates the glamour renderer.
func (m *ViewportModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(styles.DarkStyleConfig),
		glamour.WithWordWrap(width),
	)
	if err == nil {
		m.renderer = r
	}
}
