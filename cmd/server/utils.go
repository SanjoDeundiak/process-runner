package main

import (
	protov1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"github.com/SanjoDeundiak/process-runner/pkg/lib"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoProcess(p *lib.Command) *protov1.Process {
	return &protov1.Process{Command: p.Command, Args: p.Args}
}

func toProtoProcessStatus(st *lib.ProcessStatus) *protov1.ProcessStatus {
	ps := &protov1.ProcessStatus{
		State:     toProtoProcessState(st.State),
		StartTime: timestamppb.New(st.StartTime),
	}
	if st.ExitCode != nil {
		v := int32(*st.ExitCode)
		ps.ExitCode = &v
	}
	if st.EndTime != nil {
		ps.EndTime = timestamppb.New(*st.EndTime)
	}
	return ps
}

func toProtoProcessState(s lib.ProcessState) protov1.ProcessState {
	switch s {
	case lib.ProcessStateRunning:
		return protov1.ProcessState_PROCESS_STATE_RUNNING
	case lib.ProcessStateStopped:
		return protov1.ProcessState_PROCESS_STATE_STOPPED
	default:
		return protov1.ProcessState_PROCESS_STATE_UNSPECIFIED
	}
}
