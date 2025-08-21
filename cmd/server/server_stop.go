package main

import (
	"context"
	"errors"
	"os"

	protov1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ProcessRunnerServiceServer) Stop(ctx context.Context, request *protov1.StopRequest) (*protov1.StopResponse, error) {
	processIdentifier := request.ProcessIdentifier

	err := s.checkOwnership(ctx, processIdentifier)

	if err != nil {
		return nil, err
	}

	res, err := s.runner.Stop(processIdentifier)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "process not found: %s", request.ProcessIdentifier)
		}
		return nil, status.Errorf(codes.Internal, "error stopping process: %v", err)
	}
	return &protov1.StopResponse{Process: toProtoProcess(res.Command), Status: toProtoProcessStatus(res.Status)}, nil
}
