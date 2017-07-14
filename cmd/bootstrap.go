package cmd

import (
	bootstrap "github.com/hootsuite/atlantis/bootstrap"
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Start atlantis server for the first time",
	RunE: withErrPrint(func(cmd *cobra.Command, args []string) error {
		return bootstrap.Start()
	}),
}

func init() {
	RootCmd.AddCommand(bootstrapCmd)
}
