package output_storage

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// helper: receive all until channel closes
func recvAllString(t *testing.T, ch <-chan []byte) string {
	t.Helper()
	var out []byte
	for b := range ch {
		out = append(out, b...)
	}
	return string(out)
}

func TestSubscribe_ConcurrentSubscribersWhileAppending(t *testing.T) {
	s := RunNewOutputStorage()

	// Prepare expected output (single appender to preserve order guarantees)
	const N = 300
	expected := make([]byte, 0, N*4)
	for i := 1; i <= N; i++ {
		expected = append(expected, []byte(fmt.Sprintf("%d\n", i))...)
	}

	// Start subscribers before appending
	const subs = 10
	chs := make([]<-chan []byte, 0, subs)
	for i := 0; i < subs; i++ {
		ch := s.Subscribe(32)
		chs = append(chs, ch)
	}

	// Appender goroutine
	go func() {
		for i := 1; i <= N; i++ {
			s.Append([]byte(fmt.Sprintf("%d\n", i)))
			// small jitter to exercise scheduling
			time.Sleep(time.Microsecond * 200)
		}
		// allow deliveries to flush
		time.Sleep(20 * time.Millisecond)
		s.Stop()
	}()

	var wg sync.WaitGroup
	wg.Add(subs)
	outs := make([]string, subs)
	for i := 0; i < subs; i++ {
		i := i
		go func() { defer wg.Done(); outs[i] = recvAllString(t, chs[i]) }()
	}
	// Wait for all subscribers with a timeout to avoid hanging tests
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for subscribers to finish")
	}

	expectedStr := string(expected)
	for i := 0; i < subs; i++ {
		if outs[i] != expectedStr {
			t.Fatalf("subscriber %d mismatch: got %d bytes, want %d", i, len(outs[i]), len(expectedStr))
		}
	}

}

func TestSubscribe_ManySubscribersCloseOnCancel(t *testing.T) {
	s := RunNewOutputStorage()

	const subs = 50
	chs := make([]<-chan []byte, 0, subs)
	for i := 0; i < subs; i++ {
		ch := s.Subscribe(1)
		chs = append(chs, ch)
	}

	// Start readers that just drain until close
	var wg sync.WaitGroup
	wg.Add(subs)
	for i := 0; i < subs; i++ {
		ch := chs[i]
		go func() {
			for range ch {
			}
			wg.Done()
		}()
	}

	// Cancel shortly; all channels should close and goroutines finish
	s.Stop()

	c := make(chan struct{})
	go func() { wg.Wait(); close(c) }()

	select {
	case <-c:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("subscribers did not close on cancel in time")
	}

}
