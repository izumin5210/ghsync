package cmd

import (
	"github.com/izumin5210/clig/pkg/clib"
	"github.com/spf13/cobra"
)

func New(io clib.IO) *cobra.Command {
	cmd := &cobra.Command{
		Use: "ghsync",
	}

	cmd.AddCommand(
		newPushCmd(),
		newCleanCmd(),
		newVersionCmd(),
	)

	cmd.SetOut(io.Out())
	cmd.SetErr(io.Err())
	cmd.SetIn(io.In())
	clib.AddLoggingFlags(cmd)

	return cmd
}
