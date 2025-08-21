package main

import (
	"context"

	protov1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ProcessRunnerServiceServer) Start(ctx context.Context, request *protov1.StartRequest) (*protov1.StartResponse, error) {
	logger.Printf("Starting process: %s %s", request.Command, request.Args)
	startResult, err := s.runner.Start(request.Command, request.Args...)
	logger.Printf("Started process: %s %s", request.Command, request.Args)

	if err != nil {
		return nil, status.Errorf(codes.Aborted, "Error starting process: %s", err)
	}

	response := &protov1.StartResponse{}
	processIdentifier := startResult.ID
	response.ProcessIdentifier = processIdentifier
	response.Status = &protov1.ProcessStatus{
		State:     protov1.ProcessState_PROCESS_STATE_RUNNING,
		ExitCode:  new(int32),
		StartTime: nil,
		EndTime:   nil,
	}

	spiffeId := extractSpiffeIdFromContext(ctx)

	if spiffeId != nil {
		s.mu.Lock()
		s.ownersMap[processIdentifier] = *spiffeId
		s.mu.Unlock()
	}

	return response, nil
}
