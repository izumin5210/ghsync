package cmd

import (
	"github.com/izumin5210/clig/pkg/clib"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ghsync",
	}

	cmd.AddCommand(
		newPushCmd(),
	)

	clib.AddLoggingFlags(cmd)

	return cmd
}
