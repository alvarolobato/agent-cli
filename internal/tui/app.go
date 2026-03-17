package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
	"github.com/alvarolobato/agent-cli/internal/tui/dashboard"
	"github.com/alvarolobato/agent-cli/internal/tui/detail"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenDashboard screen = iota
	screenInputDetail
	screenProcessorDetail
	screenOutputDetail
	screenErrors
	screenRawConfig
)

type tickMsg time.Time

// Model is the root Bubbletea model for agent-cli.
type Model struct {
	screen      screen
	dash        dashboard.Model
	live        bool
	refresh     time.Duration
	lastUpdated time.Time
	refreshes   int

	inputDetail     detail.InputDetail
	processorDetail detail.ProcessorDetail
	outputDetail    detail.OutputDetail
	errorsView      detail.ErrorsView
	rawConfig       detail.RawConfig
}

// NewModel builds the TUI root model.
func NewModel(live bool, refresh time.Duration, pipe *pipeline.Pipeline) Model {
	interval := clampRefreshInterval(refresh)
	initiallyLive := live || refresh > 0
	now := time.Now().UTC()
	rawConfig := detail.NewRawConfig(buildRawConfigSnapshot(pipe))
	return Model{
		screen:          screenDashboard,
		dash:            dashboard.NewModel(pipe),
		live:            initiallyLive,
		refresh:         interval,
		lastUpdated:     now,
		inputDetail:     detail.NewInputDetail(firstNode(pipe, "input", "receiver")),
		processorDetail: detail.NewProcessorDetail(firstNode(pipe, "processor")),
		outputDetail:    detail.NewOutputDetail(firstNode(pipe, "output", "exporter")),
		errorsView:      detail.NewErrorsView(pipe.Metadata),
		rawConfig:       rawConfig,
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
		case "up", "left":
			if m.screen != screenDashboard {
				return m, nil
			}
			m.dash.MoveUp()
		case "down", "right":
			if m.screen != screenDashboard {
				return m, nil
			}
			m.dash.MoveDown()
		case "r":
			m.lastUpdated = time.Now().UTC()
			m.refreshes++
		case "enter":
			if m.screen != screenDashboard {
				return m, nil
			}
			switch m.dash.SelectedColumn() {
			case 0:
				m.screen = screenInputDetail
			case 1:
				m.screen = screenProcessorDetail
			case 2:
				m.screen = screenOutputDetail
			}
		case "e":
			m.screen = screenErrors
		case "c":
			m.screen = screenRawConfig
		case "esc", "b":
			m.screen = screenDashboard
		case "l":
			m.live = !m.live
			if m.live {
				return m, tea.Tick(m.refresh, func(t time.Time) tea.Msg { return tickMsg(t) })
			}
		}
	case tickMsg:
		if m.live {
			m.lastUpdated = time.Time(msg).UTC()
			m.refreshes++
			return m, tea.Tick(m.refresh, func(t time.Time) tea.Msg { return tickMsg(t) })
		}
	}

	if m.screen == screenRawConfig {
		updated, cmd := m.rawConfig.Update(msg)
		m.rawConfig = updated
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	header := "agent-cli TUI"
	if m.live {
		header = fmt.Sprintf("%s (live every %s)", header, m.refresh)
	}
	header = fmt.Sprintf("%s\nLast updated: %s | refreshes: %d", header, m.lastUpdated.Format(time.RFC3339), m.refreshes)

	switch m.screen {
	case screenDashboard:
		return header + "\n\n" + m.dash.View() + "\n\n(Enter detail, e errors, c raw config, l toggle live, q quit)"
	case screenInputDetail:
		return header + "\n\n" + m.inputDetail.View()
	case screenProcessorDetail:
		return header + "\n\n" + m.processorDetail.View()
	case screenOutputDetail:
		return header + "\n\n" + m.outputDetail.View()
	case screenErrors:
		return header + "\n\n" + m.errorsView.View()
	case screenRawConfig:
		return header + "\n\n" + m.rawConfig.View()
	default:
		return header + "\n\nUnknown screen"
	}
}

func clampRefreshInterval(refresh time.Duration) time.Duration {
	if refresh <= 0 {
		return 5 * time.Second
	}
	if refresh < time.Second {
		return time.Second
	}
	return refresh
}

func firstNode(pipe *pipeline.Pipeline, kinds ...string) pipeline.Node {
	if pipe == nil {
		return pipeline.Node{}
	}
	lookup := map[string]struct{}{}
	for _, kind := range kinds {
		lookup[kind] = struct{}{}
	}
	for _, node := range pipe.Nodes {
		if _, ok := lookup[node.Kind]; ok {
			return node
		}
	}
	return pipeline.Node{}
}

func buildRawConfigSnapshot(pipe *pipeline.Pipeline) string {
	if pipe == nil {
		return "pipeline: {}\n"
	}
	var b strings.Builder
	b.WriteString("pipeline:\n")
	for _, node := range pipe.Nodes {
		b.WriteString("  - id: ")
		b.WriteString(node.ID)
		b.WriteString("\n    kind: ")
		b.WriteString(node.Kind)
		b.WriteString("\n    status: ")
		b.WriteString(string(node.Status))
		b.WriteString("\n")
	}
	return b.String()
}
