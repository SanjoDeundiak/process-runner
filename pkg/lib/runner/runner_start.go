package runner

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/SanjoDeundiak/process-runner/pkg/lib"
	"github.com/SanjoDeundiak/process-runner/pkg/lib/output_storage"
)

type StartResult struct {
	ID     string
	pid    int
	Status *lib.ProcessStatus
}

// Start starts a new process, returning its generated identifier and initial status.
func (runner *Runner) Start(command string, args ...string) (*StartResult, error) {
	if command == "" {
		return nil, errors.New("command is required")
	}
	processId := lib.NewID()
	workDir := filepath.Join(runner.baseDir, processId)
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return nil, err
	}
	// Note that this folder is not removed

	cmd := exec.Command(command, args...)
	cmd.Dir = workDir

	sysProcAttr, err := GetSysProcAttr(processId)
	if err != nil {
		return nil, err
	}
	cmd.SysProcAttr = sysProcAttr.Raw

	stdout := output_storage.RunNewOutputStorage()
	stderr := output_storage.RunNewOutputStorage()

	// cmd.Stdin is left nil, so it will use /dev/null
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	processEntry := &processEntry{
		id:      processId,
		command: lib.Command{Command: command, Args: append([]string(nil), args...)},
		cmd:     cmd,
		workDir: workDir,
		state:   lib.ProcessStateRunning,
		start:   time.Now(),
		stdout:  stdout,
		stderr:  stderr,
	}

	// Start process
	logger.Printf("Starting process %s", processId)
	if err := cmd.Start(); err != nil {
		logger.Printf("Failed to start process %s: %v", processId, err)
		return nil, err
	}

	if sysProcAttr.File != nil {
		_ = sysProcAttr.File.Close()
	}

	processEntry.pid = cmd.Process.Pid

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Waiter
	go func() {
		logger.Printf("Waiting for process %s to finish\n", processId)

		err := <-done

		if err != nil {
			logger.Printf("Process %s finished with err: %s\n", processId, err)
		} else {
			logger.Printf("Process %s finished without error\n", processId)
		}

		stdout.Stop()
		stderr.Stop()

		processEntry.mu.Lock()
		defer func() {
			processEntry.mu.Unlock()
		}()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				code := exitErr.ExitCode()
				processEntry.exitCode = &code
			} else {
				// Non-exit error, leave exitCode nil
			}
		} else {
			code := 0
			processEntry.exitCode = &code
		}
		now := time.Now()
		processEntry.end = &now
		processEntry.state = lib.ProcessStateStopped
		logger.Printf("Process %s calling cancel", processId)

		// Cleanup platform-specific resources
		_ = CleanupCgroup(processId)
	}()

	runner.mu.Lock()
	runner.processes[processId] = processEntry
	runner.mu.Unlock()

	status := processEntry.lockAndGetStatus()

	return &StartResult{ID: processId, pid: cmd.Process.Pid, Status: &status}, nil
}
