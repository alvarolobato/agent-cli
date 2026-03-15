package tui

import (
	"fmt"
	"time"

	"github.com/alvarolobato/agent-cli/internal/tui/dashboard"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenDashboard screen = iota
)

type tickMsg time.Time

// Model is the root Bubbletea model for agent-cli.
type Model struct {
	screen  screen
	dash    dashboard.Model
	live    bool
	refresh time.Duration
}

// NewModel builds the TUI root model.
func NewModel(live bool, refresh time.Duration) Model {
	return Model{
		screen:  screenDashboard,
		dash:    dashboard.NewModel(),
		live:    live,
		refresh: refresh,
	}
}

func (m Model) Init() tea.Cmd {
	if m.live {
		return tea.Tick(m.refresh, func(t time.Time) tea.Msg { return tickMsg(t) })
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up":
			m.dash.MoveUp()
		case "down":
			m.dash.MoveDown()
		case "r":
			// Placeholder refresh trigger; data fetch will be wired in later phases.
		}
	case tickMsg:
		if m.live {
			return m, tea.Tick(m.refresh, func(t time.Time) tea.Msg { return tickMsg(t) })
		}
	}

	return m, nil
}

func (m Model) View() string {
	header := "agent-cli TUI"
	if m.live {
		header = fmt.Sprintf("%s (live every %s)", header, m.refresh)
	}
	switch m.screen {
	case screenDashboard:
		return header + "\n\n" + m.dash.View() + "\n\n(q to quit, r to refresh)"
	default:
		return header + "\n\nUnknown screen"
	}
}
