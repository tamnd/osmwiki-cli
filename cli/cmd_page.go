package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/osmwiki-cli/osmwiki"
)

func (a *App) pageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "page <title>",
		Short: "Get OpenStreetMap wiki page summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			detail, err := a.client.GetPage(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			if detail == nil {
				return codeError(exitNoData, nil)
			}
			return a.render([]osmwiki.PageDetail{*detail})
		},
	}
}
