package lib

import "time"

// ProcessState mirrors high-level states similar to proto/v1.ProcessState
// It's intentionally minimal; more states can be added later.
type ProcessState int

const (
	ProcessStateUnspecified ProcessState = iota
	ProcessStateRunning
	ProcessStateStopped
)

// Command captures command metadata used to start a process.
type Command struct {
	Command string
	Args    []string
}

// ProcessStatus captures runtime state and timestamps.
type ProcessStatus struct {
	State     ProcessState
	ExitCode  *int
	StartTime time.Time
	EndTime   *time.Time
}
