package main

import (
	"io"
	"log"
	"sync"

	protov1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"github.com/SanjoDeundiak/process-runner/pkg/lib/runner"
)

var logger = log.New(io.Discard, "server: ", log.LstdFlags)

type ProcessRunnerServiceServer struct {
	protov1.UnimplementedProcessRunnerServiceServer
	runner    *runner.Runner
	mu        sync.RWMutex
	ownersMap map[string]string
}

func NewProcessRunnerServiceServer() (*ProcessRunnerServiceServer, error) {
	r, err := runner.NewRunner()
	if err != nil {
		return nil, err
	}

	return &ProcessRunnerServiceServer{
		runner:    r,
		ownersMap: make(map[string]string),
	}, nil
}
