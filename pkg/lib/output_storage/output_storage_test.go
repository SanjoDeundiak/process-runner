package output_storage

import (
	"fmt"
	"testing"
	"time"
)

func TestNewOutputStorage_Empty(t *testing.T) {
	s := RunNewOutputStorage()
	defer s.Stop()

	cnt := 0
	s.ForEach(func(b []byte) bool {
		cnt++
		return true
	})
	if cnt != 0 {
		t.Fatalf("expected 0 items, got %d", cnt)
	}
	if got := s.Bytes(); len(got) != 0 {
		t.Fatalf("expected empty bytes, got %q", string(got))
	}
}

func TestAppendAndForEach_OrderAndEarlyStop(t *testing.T) {
	s := RunNewOutputStorage()
	defer s.Stop()
	s.Append([]byte("a"))
	s.Append([]byte("b"))
	s.Append([]byte("c"))

	var got []string
	s.ForEach(func(b []byte) bool {
		got = append(got, string(b))
		return true
	})
	want := []string{"a", "b", "c"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("order mismatch: got=%v want=%v", got, want)
	}

	// Early stop after two elements
	got = nil
	calls := 0
	s.ForEach(func(b []byte) bool {
		calls++
		got = append(got, string(b))
		return calls < 2
	})
	if calls != 2 || fmt.Sprint(got) != fmt.Sprint([]string{"a", "b"}) {
		t.Fatalf("early stop failed: calls=%d got=%v", calls, got)
	}
}

func TestBytes_Concatenation(t *testing.T) {
	s := RunNewOutputStorage()
	defer s.Stop()
	s.Append([]byte("hello "))
	s.Append([]byte("world"))
	if got, want := string(s.Bytes()), "hello world"; got != want {
		t.Fatalf("Bytes mismatch: got=%q want=%q", got, want)
	}
}

func TestNilReceiverSafety(t *testing.T) {
	// Methods should be safe on a nil receiver per implementation.
	var s *OutputStorage

	// ForEach with nil receiver and nil iter should not panic.
	s.ForEach(nil)

	// ForEach with function should not be called at all
	called := false
	s.ForEach(func(b []byte) bool {
		called = true
		return true
	})
	if called {
		t.Fatalf("ForEach should not invoke iter for nil receiver")
	}

	// Append on nil should be a no-op and not panic
	s.Append([]byte("x"))

	// Bytes on nil should return empty per current implementation path
	if got := s.Bytes(); len(got) != 0 {
		t.Fatalf("expected empty bytes from nil receiver, got %q", string(got))
	}
}

func TestAppendStoresSliceByReference(t *testing.T) {
	s := RunNewOutputStorage()
	defer s.Stop()
	data := []byte("abc")
	s.Append(data)
	data[0] = 'z'
	if got := string(s.Bytes()); got != "zbc" {
		t.Fatalf("expected storage to reflect slice mutation, got %q", got)
	}
}

func TestSubscribe_DeliversExistingItemsInOrder(t *testing.T) {
	s := RunNewOutputStorage()
	defer s.Stop()
	s.Append([]byte("a"))
	s.Append([]byte("b"))
	s.Append([]byte("c"))

	ch := s.Subscribe(3)

	if v, ok := recvWithTimeout[[]byte](t, ch, 200*time.Millisecond); !ok || string(v) != "a" {
		t.Fatalf("expected first item 'a', ok=%v v=%q", ok, string(v))
	}
	if v, ok := recvWithTimeout[[]byte](t, ch, 200*time.Millisecond); !ok || string(v) != "b" {
		t.Fatalf("expected second item 'b', ok=%v v=%q", ok, string(v))
	}
	if v, ok := recvWithTimeout[[]byte](t, ch, 200*time.Millisecond); !ok || string(v) != "c" {
		t.Fatalf("expected third item 'c', ok=%v v=%q", ok, string(v))
	}

	// No further messages should arrive without new appends
	assertNoRecv[[]byte](t, ch, 50*time.Millisecond)
}

func TestSubscribe_ChannelClosesOnCancel(t *testing.T) {
	s := RunNewOutputStorage()
	s.Append([]byte("x"))

	ch := s.Subscribe(1)

	if v, ok := recvWithTimeout[[]byte](t, ch, 200*time.Millisecond); !ok || string(v) != "x" {
		t.Fatalf("expected initial item 'x', ok=%v v=%q", ok, string(v))
	}

	// Start a goroutine to wait for channel close
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()

	// Cancel the context and expect the subscription channel to close
	s.Stop()

	select {
	case <-done:
		// closed as expected
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("subscription channel did not close after context cancellation")
	}
}
