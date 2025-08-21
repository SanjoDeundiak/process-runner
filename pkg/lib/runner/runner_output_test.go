package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/SanjoDeundiak/process-runner/pkg/lib"
)

// readAll collects all bytes from a subscription channel until it closes.
func readAll(t *testing.T, ch <-chan []byte) string {
	t.Helper()
	var out []byte
	for b := range ch {
		out = append(out, b...)
	}
	return string(out)
}

func TestStdOut_MultipleSubscribers(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	// Emit a few lines with small delays to allow subscribers to attach and stream
	// shell is available in test envs
	res, err := r.Start("sh", "-c", "for i in 1 2 3 4 5; do echo $i; sleep 0.03; done")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Two subscribers to Output
	ch1, _, err := r.Output(res.ID)
	if err != nil {
		t.Fatalf("Output#1 failed: %v", err)
	}
	ch2, _, err := r.Output(res.ID)
	if err != nil {
		t.Fatalf("Output#2 failed: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var s1, s2 string
	go func() { defer wg.Done(); s1 = readAll(t, ch1) }()
	go func() { defer wg.Done(); s2 = readAll(t, ch2) }()

	// Wait until process stops and both channels close
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		st, err := r.Status(res.ID)
		if err != nil {
			t.Fatalf("Status error: %v", err)
		}
		if st.Status.State == lib.ProcessStateStopped && st.Status.ExitCode != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()

	expected := "1\n2\n3\n4\n5\n"
	if s1 != expected || s2 != expected {
		t.Fatalf("subscribers mismatch:\nch1=%q\nch2=%q\nwant=%q", s1, s2, expected)
	}
}

func TestStdOut_LateSubscriberReceivesBacklog(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	res, err := r.Start("sh", "-c", "for i in 1 2 3 4; do echo $i; sleep 0.05; done")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// First subscriber starts immediately
	ch1, _, err := r.Output(res.ID)
	if err != nil {
		t.Fatalf("Output#1 failed: %v", err)
	}

	// Wait until at least two lines are likely produced
	time.Sleep(120 * time.Millisecond)

	// Late subscriber should receive full backlog from the beginning per OutputStorage semantics
	ch2, _, err := r.Output(res.ID)
	if err != nil {
		t.Fatalf("Output#2 failed: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var s1, s2 string
	go func() { defer wg.Done(); s1 = readAll(t, ch1) }()
	go func() { defer wg.Done(); s2 = readAll(t, ch2) }()

	// Wait for completion
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		st, err := r.Status(res.ID)
		if err != nil {
			t.Fatalf("Status error: %v", err)
		}
		if st.Status.State == lib.ProcessStateStopped && st.Status.ExitCode != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()

	expected := "1\n2\n3\n4\n"
	if s1 != expected || s2 != expected {
		t.Fatalf("late subscriber mismatch:\nch1=%q\nch2=%q\nwant=%q", s1, s2, expected)
	}
}

func TestStdOut_ConcurrentSubscribersAndStatus(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	// Print 100 lines rapidly
	res, err := r.Start("sh", "-c", "i=1; while [ $i -le 100 ]; do echo $i; i=$((i+1)); done;")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	const subs = 5
	chs := make([]<-chan []byte, 0, subs)
	for i := 0; i < subs; i++ {
		ch, _, err := r.Output(res.ID)
		if err != nil {
			t.Fatalf("Output(%d) failed: %v", i, err)
		}
		chs = append(chs, ch)
	}

	var wg sync.WaitGroup
	wg.Add(subs)
	outs := make([]string, subs)
	for i := 0; i < subs; i++ {
		i := i
		go func() { defer wg.Done(); outs[i] = readAll(t, chs[i]) }()
	}

	// Wait for all subscribers to finish with a timeout to avoid hangs
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for subscribers to finish at %v", time.Now().Format("15:04:05"))
	}

	// Basic validation: all non-empty and equal
	for i := 0; i < subs; i++ {
		if outs[i] == "" {
			t.Fatalf("subscriber %d got empty output", i)
		}
		if outs[i] != outs[0] {
			t.Fatalf("subscriber outputs differ: %q vs %q", outs[i], outs[0])
		}
	}

	// Expect last line to be 100
	if got, want := outs[0][len(outs[0])-4:], "100\n"; !containsSuffix(outs[0], want) {
		t.Fatalf("unexpected output ending: got tail=%q want=%q", got, want)
	}
}

func containsSuffix(s, suf string) bool {
	if len(s) < len(suf) {
		return false
	}
	return s[len(s)-len(suf):] == suf
}

func TestStdOut_NoOutputChannelsClose(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	// Run a command that produces no output and exits immediately
	res, err := r.Start("sh", "-c", ":")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	outCh, errCh, err := r.Output(res.ID)
	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}
	
	done := make(chan struct{})
	var outS, errS string
	go func() {
		outS = readAll(t, outCh)
		errS = readAll(t, errCh)
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatalf("channels did not close for no-output process")
	}

	if outS != "" || errS != "" {
		t.Fatalf("expected no data, got stdout=%q stderr=%q", outS, errS)
	}
}
