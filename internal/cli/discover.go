package cli

import (
	"context"
	"strconv"
	"strings"

	"github.com/alvarolobato/agent-cli/internal/discovery"
	"github.com/spf13/cobra"
)

func newDiscoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Discover local agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			agents, err := discovery.NewOrchestrator().DiscoverDetailed(ctx)
			if err != nil {
				return err
			}

			cmd.Printf("Found %d agent(s):\n", len(agents))
			for i, a := range agents {
				configPath := a.ConfigPath
				if strings.TrimSpace(configPath) == "" {
					configPath = "(config not found)"
				}
				pidLabel := "PID n/a"
				if a.PID > 0 {
					pidLabel = "PID " + strconv.Itoa(a.PID)
				}
				cmd.Printf("  %d. %s (%s) - %s [%s]\n", i+1, a.AgentType, pidLabel, configPath, a.Source)
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
