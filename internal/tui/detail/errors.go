package detail

import (
	"fmt"
	"sort"
	"strings"
)

type ErrorsView struct {
	Warnings []string
}

func NewErrorsView(metadata map[string]string) ErrorsView {
	items := make([]string, 0)
	for key, value := range metadata {
		if !strings.Contains(strings.ToLower(key), "warning") {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}
	sort.Strings(items)
	return ErrorsView{Warnings: items}
}

func (m ErrorsView) View() string {
	if len(m.Warnings) == 0 {
		return "Errors and warnings\n\nNo issues detected.\n\nEsc/b to go back"
	}
	return fmt.Sprintf("Errors and warnings\n\n- %s\n\nEsc/b to go back", strings.Join(m.Warnings, "\n- "))
}
