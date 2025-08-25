package main

import (
	"context"
	"errors"
	"os"

	protov1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ProcessRunnerServiceServer) Status(ctx context.Context, request *protov1.StatusRequest) (*protov1.StatusResponse, error) {
	processIdentifier := request.ProcessIdentifier

	err := s.checkOwnership(ctx, processIdentifier)

	if err != nil {
		return nil, err
	}

	statusResult, err := s.runner.Status(processIdentifier)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "process not found: %s", request.ProcessIdentifier)
		}
		return nil, status.Errorf(codes.Internal, "error getting status: %v", err)
	}
	resp := &protov1.StatusResponse{
		Process: toProtoProcess(statusResult.Command),
		Status:  toProtoProcessStatus(statusResult.Status),
	}
	return resp, nil
}
