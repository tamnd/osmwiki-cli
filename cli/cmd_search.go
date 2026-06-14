package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search OpenStreetMap wiki pages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit := a.effectiveLimit(20)
			pages, err := a.client.Search(cmd.Context(), args[0], limit)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(pages, len(pages))
		},
	}
	return cmd
}
