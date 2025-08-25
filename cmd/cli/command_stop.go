package main

import (
	"context"
	"fmt"
	"os"
	"time"

	apiv1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <process_id>",
		Short: "Stop a process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			processID := args[0]
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			conn, err := dial(ctx)
			if err != nil {
				return err
			}
			defer conn.Close()

			client := apiv1.NewProcessRunnerServiceClient(conn)
			resp, err := client.Stop(ctx, &apiv1.StopRequest{ProcessIdentifier: processID})
			if err != nil {
				if grpcCode(err) == codes.PermissionDenied {
					_, _ = fmt.Fprintln(os.Stderr, "Forbidden. Only the creator of the process can stop it.")
					return nil
				}
				return err
			}
			// Print the status and process returned by Stop directly
			printStatusTable(processID, resp.GetStatus(), resp.GetProcess())
			return nil
		},
	}
	return cmd
}
