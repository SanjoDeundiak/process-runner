package runner

import (
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/SanjoDeundiak/process-runner/pkg/lib"
	"github.com/SanjoDeundiak/process-runner/pkg/lib/output_storage"
)

var logger = log.New(io.Discard, "", log.LstdFlags)

// Runner manages processes started by this library.
type Runner struct {
	mu        sync.RWMutex
	processes map[string]*processEntry
	baseDir   string
}

type processEntry struct {
	id      string
	command lib.Command
	cmd     *exec.Cmd
	workDir string

	// status fields
	mu       sync.RWMutex
	state    lib.ProcessState
	exitCode *int
	start    time.Time
	end      *time.Time
	// output buffer (full replay)
	stdout *output_storage.OutputStorage
	stderr *output_storage.OutputStorage
	pid    int
}

// NewRunner creates a new Runner.
func NewRunner() (*Runner, error) {
	baseDir, err := os.MkdirTemp("", "prn-*")
	if err != nil {
		return nil, err
	}

	return &Runner{processes: make(map[string]*processEntry), baseDir: baseDir}, nil
}
