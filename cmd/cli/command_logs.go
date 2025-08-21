package main

import (
	"context"
	"io"
	"os"

	apiv1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <process_id>",
		Short: "Stream logs (stdout/stderr) from the beginning",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			processID := args[0]
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			conn, err := dial(ctx)
			if err != nil {
				return err
			}
			defer conn.Close()

			client := apiv1.NewProcessRunnerServiceClient(conn)
			stream, err := client.GetOutput(ctx, &apiv1.GetOutputRequest{ProcessIdentifier: processID})
			if err != nil {
				return err
			}
			for {
				msg, err := stream.Recv()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}

				var w io.Writer = nil
				switch msg.GetType() {
				case apiv1.GetOutputResponse_TYPE_STDOUT:
					w = os.Stdout
				case apiv1.GetOutputResponse_TYPE_STDERR:
					w = os.Stderr
				}

				if w != nil {
					_, werr := w.Write(msg.GetData())
					if werr != nil {
						return werr
					}
				}
			}
		},
	}
	return cmd
}
