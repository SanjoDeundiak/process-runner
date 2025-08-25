package main

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "prn",
		Short:         "Process Runner CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newStartCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newStopCmd())
	root.AddCommand(newLogsCmd())

	return root
}
