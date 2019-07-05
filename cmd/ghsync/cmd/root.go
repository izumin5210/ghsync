package cmd

import "github.com/spf13/cobra"

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ghsync",
	}

	cmd.AddCommand(
		newPushCmd(),
	)

	return cmd
}
