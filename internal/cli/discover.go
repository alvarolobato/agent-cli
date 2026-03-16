package cli

import (
	"context"
	"strings"

	"github.com/alvarolobato/agent-cli/internal/discovery"
	"github.com/spf13/cobra"
)

func newDiscoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Discover local agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			agents, err := discovery.NewOrchestrator().DiscoverDetailed(context.Background())
			if err != nil {
				return err
			}

			cmd.Printf("Found %d agent(s):\n", len(agents))
			for i, a := range agents {
				configPath := a.ConfigPath
				if strings.TrimSpace(configPath) == "" {
					configPath = "(config not found)"
				}
				cmd.Printf("  %d. %s (PID %d) - %s [%s]\n", i+1, a.AgentType, a.PID, configPath, a.Source)
				if len(a.Children) == 0 {
					continue
				}
				cmd.Println("     Children:")
				for _, child := range a.Children {
					role := child.Role
					if strings.TrimSpace(role) == "" {
						role = child.Name
					}
					cmd.Printf("       - %s (PID %d)\n", role, child.PID)
				}
			}
			return nil
		},
	}
}
