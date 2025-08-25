package lib

import "time"

// ProcessState mirrors high-level states similar to proto/v1.ProcessState
// It's intentionally minimal; more states can be added later.
type ProcessState int

const (
	ProcessStateUnspecified ProcessState = iota
	ProcessStateRunning
	ProcessStateSleeping // not used currently
	ProcessStateStopped
	ProcessStateZombie
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

// OutputType indicates which stream produced the data.
type OutputType int

const (
	OutputTypeUnspecified OutputType = iota
	OutputTypeStdout
	OutputTypeStderr
)

// OutputChunk is a piece of output with type and data.
type OutputChunk struct {
	Type OutputType
	Data []byte
}

// Subscription provides full replay of a process's output and then tails until completion.
// Call Close() to stop receiving.
type Subscription struct {
	C         <-chan OutputChunk
	CloseFunc func()
}

func (s *Subscription) Close() {
	if s != nil && s.CloseFunc != nil {
		s.CloseFunc()
	}
}
