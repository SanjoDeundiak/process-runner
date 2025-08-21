package main

import (
	"errors"
	"os"

	protov1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ProcessRunnerServiceServer) GetOutput(request *protov1.GetOutputRequest, streaming grpc.ServerStreamingServer[protov1.GetOutputResponse]) error {
	stdout, stderr, err := s.runner.Output(request.ProcessIdentifier)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status.Errorf(codes.NotFound, "process not found: %s", request.ProcessIdentifier)
		}
		return status.Errorf(codes.Internal, "error subscribing to output: %v", err)
	}

	ctx := streaming.Context()
	for {
		if stdout == nil && stderr == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case chunk, ok := <-stdout:
			if !ok {
				stdout = nil
				continue
			}

			if err := streaming.Send(&protov1.GetOutputResponse{Type: protov1.GetOutputResponse_TYPE_STDOUT, Data: chunk}); err != nil {
				return err
			}
		case chunk, ok := <-stderr:
			if !ok {
				stderr = nil
				continue
			}

			if err := streaming.Send(&protov1.GetOutputResponse{Type: protov1.GetOutputResponse_TYPE_STDERR, Data: chunk}); err != nil {
				return err
			}
		}
	}
}
