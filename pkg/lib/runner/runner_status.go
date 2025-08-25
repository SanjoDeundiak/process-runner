package runner

import (
	"os"

	"github.com/SanjoDeundiak/process-runner/pkg/lib"
)

type StatusResult struct {
	Command *lib.Command
	Status  *lib.ProcessStatus
}

// Status returns the current process and status by identifier.
func (runner *Runner) Status(id string) (*StatusResult, error) {
	pe, err := runner.getProcess(id)
	if err != nil {
		return nil, err
	}

	status := pe.lockAndGetStatus()
	result := StatusResult{
		Command: &pe.command,
		Status:  &status,
	}

	return &result, nil
}

func (runner *Runner) getProcess(id string) (*processEntry, error) {
	runner.mu.RLock()
	pe := runner.processes[id]
	runner.mu.RUnlock()
	if pe == nil {
		return nil, os.ErrNotExist
	}
	return pe, nil
}

func (processEntry *processEntry) lockAndGetStatus() lib.ProcessStatus {
	processEntry.mu.RLock()
	defer processEntry.mu.RUnlock()

	st := lib.ProcessStatus{State: processEntry.state, StartTime: processEntry.start}
	if processEntry.exitCode != nil {
		st.ExitCode = new(int)
		*st.ExitCode = *processEntry.exitCode
	}
	if processEntry.end != nil {
		t := *processEntry.end
		st.EndTime = &t
	}
	return st
}
