package cmd

import (
	"fmt"
	"runtime"

	"github.com/izumin5210/ghsync"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print the version information",
		Long:          "Print the version information",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "ghsync %s (%s %s/%s)\n",
				ghsync.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}
