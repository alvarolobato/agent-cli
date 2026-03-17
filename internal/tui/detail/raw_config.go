package detail

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type RawConfig struct {
	viewport viewport.Model
}

func NewRawConfig(raw string) RawConfig {
	vp := viewport.New(100, 18)
	vp.SetContent(colorizeYAML(raw))
	return RawConfig{viewport: vp}
}

func (m RawConfig) Update(msg tea.Msg) (RawConfig, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m RawConfig) View() string {
	return "Raw config\n\n" + m.viewport.View() + "\n\nEsc/b to go back"
}

func colorizeYAML(raw string) string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		lines[i] = keyStyle.Render(key) + ":" + valStyle.Render(value)
	}
	return strings.Join(lines, "\n")
}
