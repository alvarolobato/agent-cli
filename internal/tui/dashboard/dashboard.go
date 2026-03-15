package dashboard

import "strings"

// Model is a placeholder dashboard screen model.
type Model struct {
	cursor int
	items  []string
}

// NewModel returns the initial dashboard state.
func NewModel() Model {
	return Model{
		items: []string{
			"Overview",
			"Pipelines",
			"Issues",
		},
	}
}

func (m *Model) MoveUp() {
	if len(m.items) == 0 {
		return
	}
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.items) - 1
	}
}

func (m *Model) MoveDown() {
	if len(m.items) == 0 {
		return
	}
	m.cursor++
	if m.cursor >= len(m.items) {
		m.cursor = 0
	}
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("Dashboard\n")
	for i, item := range m.items {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		b.WriteString(prefix + item + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
