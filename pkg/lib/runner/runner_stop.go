package runner

import (
	"syscall"
	"time"

	"github.com/SanjoDeundiak/process-runner/pkg/lib"
)

// StopResult returns process info and its final status after Stop.
type StopResult struct {
	Command *lib.Command
	Status  *lib.ProcessStatus
}

// Stop stops the process by identifier and returns final status (or current if already stopped).
func (runner *Runner) Stop(id string) (*StopResult, error) {
	pe, err := runner.getProcess(id)
	if err != nil {
		return nil, err
	}
	res := StopResult{Command: &pe.command}
	pe.mu.RLock()
	alreadyStopped := pe.state == lib.ProcessStateStopped || pe.end != nil
	pe.mu.RUnlock()
	if alreadyStopped {
		st := pe.lockAndGetStatus()
		res.Status = &st
		return &res, nil
	}
	// Best-effort platform-specific kill: prefer cgroup kill on Linux, else kill process group
	succeeded, err := KillCgroup(id)
	if err != nil {
		return nil, err
	}

	if !succeeded {
		// Kill the process group (negative PID means process group)
		_ = syscall.Kill(-pe.pid, syscall.SIGKILL)
	}
	// Return status after it transitions; small wait loop
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		st := pe.lockAndGetStatus()
		res.Status = &st
		if st.State == lib.ProcessStateStopped {
			return &res, nil
		}
		time.Sleep(10 * time.Millisecond)
	}

	st := pe.lockAndGetStatus()
	res.Status = &st

	return &res, nil
}
