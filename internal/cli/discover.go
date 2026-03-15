package cli

import (
	"context"
	"fmt"

	"github.com/alvarolobato/agent-cli/internal/discovery"
	"github.com/spf13/cobra"
)

func newDiscoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Discover local agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			agents, err := discovery.NewOrchestrator().Discover(context.Background())
			if err != nil {
				return err
			}

			cmd.Printf("Found %d agent(s)\n", len(agents))
			for i, a := range agents {
				cmd.Println(fmt.Sprintf("%d. %s (%s)", i+1, a.ID(), a.Type()))
			}
			return nil
		},
	}
}
