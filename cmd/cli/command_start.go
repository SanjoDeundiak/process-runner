package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	apiv1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start -- <command> [args...]",
		Short: "Start a new process",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("command to execute is required; use -- to separate CLI flags from the command")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			conn, err := dial(ctx)
			if err != nil {
				return err
			}
			defer conn.Close()

			client := apiv1.NewProcessRunnerServiceClient(conn)
			resp, err := client.Start(ctx, &apiv1.StartRequest{Command: args[0], Args: args[1:]})
			if err != nil {
				return err
			}
			// Print only process ID as per design
			fmt.Println(resp.GetProcessIdentifier())
			return nil
		},
	}
	return cmd
}
