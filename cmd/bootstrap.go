package cmd

import (
	"github.com/hootsuite/atlantis/bootstrap"
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Start a guided tour of Atlantis",
	RunE: withErrPrint(func(cmd *cobra.Command, args []string) error {
		return bootstrap.Start()
	}),
}

func init() {
	RootCmd.AddCommand(bootstrapCmd)
}
