package output_storage

import (
	"testing"
	"time"
)

// helper: receive with timeout
func recvWithTimeout[T any](t *testing.T, ch <-chan T, d time.Duration) (T, bool) {
	t.Helper()
	var zero T
	select {
	case v, ok := <-ch:
		return v, ok
	case <-time.After(d):
		return zero, false
	}
}

// helper: assert no receive within duration
func assertNoRecv[T any](t *testing.T, ch <-chan T, d time.Duration) {
	t.Helper()
	if v, ok := recvWithTimeout(t, ch, d); ok {
		t.Fatalf("unexpected receive: %v", v)
	}
}

func TestBroadcaster_SingleSubscriberReceives(t *testing.T) {
	b := RunNewBroadcaster[string]()

	// Subscribe by pushing our channel into the internal queue (Subscribe returns send-only).
	ch, err := b.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	b.Publish("hello")

	if v, ok := recvWithTimeout(t, ch, 200*time.Millisecond); !ok || v != "hello" {
		t.Fatalf("expected to receive 'hello', got ok=%v val=%q", ok, v)
	}

	b.Stop()
}

func TestBroadcaster_MultipleSubscribersReceive(t *testing.T) {
	b := RunNewBroadcaster[int]()

	// Register first subscriber and ensure the goroutine processed it.
	ch1, err := b.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	b.Publish(1)
	if v, ok := recvWithTimeout(t, ch1, 200*time.Millisecond); !ok || v != 1 {
		t.Fatalf("ch1 did not receive initial message, ok=%v v=%d", ok, v)
	}

	// Now register second subscriber.
	ch2, err := b.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Broadcast again; both should see it.
	b.Publish(2)

	if v, ok := recvWithTimeout(t, ch1, 200*time.Millisecond); !ok || v != 2 {
		t.Fatalf("ch1 did not receive broadcast 2, ok=%v v=%d", ok, v)
	}
	if v, ok := recvWithTimeout(t, ch2, 200*time.Millisecond); !ok || v != 2 {
		t.Fatalf("ch2 did not receive broadcast 2, ok=%v v=%d", ok, v)
	}

	b.Stop()
}

func TestBroadcaster_NonBlockingSlowSubscriber(t *testing.T) {
	b := RunNewBroadcaster[int]()

	// Slow subscriber with a full buffer simulating being behind
	slow, err := b.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	// Pre-fill to make it full so broadcaster should drop this and deliver the latest
	slow <- -1
	// Fast subscriber that can capture the broadcast
	fast, err := b.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Publish a value; for 'slow' the oldest should be dropped and latest delivered;
	// 'fast' should receive it promptly.
	b.Publish(42)

	// Allow the message to go through
	time.Sleep(10 * time.Millisecond)

	// fast should receive
	if v, ok := recvWithTimeout(t, fast, 200*time.Millisecond); !ok || v != 42 {
		t.Fatalf("fast did not receive 42, ok=%v v=%d", ok, v)
	}

	// slow should receive the latest (42), not block or keep the old value
	if v, ok := recvWithTimeout(t, slow, 200*time.Millisecond); !ok || v != 42 {
		t.Fatalf("slow did not receive latest 42, ok=%v v=%d", ok, v)
	}

	b.Stop()
}

func TestBroadcaster_Unsubscribe(t *testing.T) {
	b := RunNewBroadcaster[int]()

	a, err := b.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	bch, err := b.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Ensure 'a' is active
	b.Publish(1)
	if v, ok := recvWithTimeout(t, a, 200*time.Millisecond); !ok || v != 1 {
		t.Fatalf("subscriber 'a' did not get initial message, ok=%v v=%d", ok, v)
	}

	<-bch

	// Unsubscribe 'a'
	b.Unsubscribe(a)

	// Publish several messages; bch should receive, 'a' should not
	for i := 0; i < 3; i++ {
		b.Publish(100 + i)
		if v, ok := recvWithTimeout(t, bch, 200*time.Millisecond); !ok || v != 100+i {
			t.Fatalf("subscriber 'bch' missed message %d, ok=%v v=%d", 100+i, ok, v)
		}
		// 'a' should not receive anything
		assertNoRecv(t, a, 50*time.Millisecond)
	}

	b.Stop()
}
