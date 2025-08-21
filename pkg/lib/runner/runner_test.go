package runner

import (
	"testing"
	"time"

	"github.com/SanjoDeundiak/process-runner/pkg/lib"
)

func getAllBytes(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()
	var all []byte
	for {
		b, ok := <-ch
		if !ok {
			break
		}
		all = append(all, b...)
	}

	return all
}

func TestStartAndOutput(t *testing.T) {
	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	res, err := runner.Start("sh", "-c", "echo out; echo err 1>&2")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	st := res.Status
	if st.State != lib.ProcessStateRunning {
		t.Fatalf("expected initial state Running, got %v", st.State)
	}
	if st.ExitCode != nil {
		t.Fatalf("expected no exit code at start")
	}
	if st.EndTime != nil {
		t.Fatalf("expected no end time at start")
	}

	stdout, stderr, err := runner.Output(res.ID)
	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}

	// Give a little time for status to update after output closes
	deadline := time.Now().Add(2 * time.Second)
	var statusResult *StatusResult
	for time.Now().Before(deadline) {
		statusResult, err = runner.Status(res.ID)
		if err != nil {
			t.Fatalf("Status error: %v", err)
		}
		if statusResult.Status.State == lib.ProcessStateStopped && statusResult.Status.ExitCode != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if statusResult.Status.State != lib.ProcessStateStopped {
		t.Fatalf("expected state Stopped, got %v", statusResult.Status.State)
	}
	if statusResult.Status.ExitCode == nil {
		t.Fatalf("expected exit code set after completion")
	}
	if *statusResult.Status.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", *statusResult.Status.ExitCode)
	}

	stdoutBytes := getAllBytes(t, stdout)
	if string(stdoutBytes) != "out\n" {
		t.Fatalf("Output: wrong value")
	}

	stderrBytes := getAllBytes(t, stderr)
	if string(stderrBytes) != "err\n" {
		t.Fatalf("Output: wrong value")
	}
}

func TestStopKillsProcess(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	res, err := r.Start("sh", "-c", "sleep 10")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if res.Status.State != lib.ProcessStateRunning {
		t.Fatalf("expected Running, got %v", res.Status.State)
	}

	// Issue Stop and wait for process to be reported as stopped
	if _, err := r.Stop(res.ID); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Poll for final state with a reasonable timeout
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		statusResult, err := r.Status(res.ID)
		if err != nil {
			t.Fatalf("Status error: %v", err)
		}
		if statusResult.Status.State == lib.ProcessStateStopped && statusResult.Status.EndTime != nil {
			// exit code may vary for SIGKILL across platforms; ensure it's set
			if statusResult.Status.ExitCode == nil {
				t.Fatalf("expected exit code set after Stop")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("process did not stop in time")
}

func TestStartInvalidCommand(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	_, err = r.Start("")
	if err == nil {
		t.Fatalf("expected error starting with empty command")
	}
}
