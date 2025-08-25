package output_storage

import (
	"io"
	"log"
	"sync/atomic"

	"github.com/google/uuid"
)

// node represents an element in the singly linked list.
// It carries a payload (byte slice) and an atomic pointer to the next node.
// The list uses a sentinel head node for simpler lock-free append logic.
type node struct {
	data []byte
	next atomic.Pointer[node]
}

var logger = log.New(io.Discard, "output_storage: ", log.LstdFlags)

// OutputStorage provides a lock-free, append-only singly linked list of byte slices.
// Appending to the end is thread-safe via atomic.Pointer usage.
// The structure is safe for concurrent Append calls and concurrent iteration.
// Reading methods (Bytes/ForEach) provide a best-effort snapshot without locks.
// If a strict snapshot is required, the caller should provide external synchronization.
type OutputStorage struct {
	head *node // sentinel head, immutable
	tail *node // last element in the list (or sentinel if empty)

	broadcaster *Broadcaster[struct{}]
}

// RunNewOutputStorage creates a new, empty OutputStorage.
func RunNewOutputStorage() *OutputStorage {
	broadcaster := RunNewBroadcaster[struct{}]()

	sentinel := &node{}
	s := &OutputStorage{
		head: sentinel,
		tail: sentinel,

		broadcaster: broadcaster,
	}

	return s
}

func (s *OutputStorage) Stop() {
	if s == nil {
		return
	}

	s.broadcaster.Stop()
}

// Append adds the provided byte slice to the end of the list in a thread-safe manner.
// Note: The slice is stored as-is; if callers may mutate the slice afterward,
// they should pass a copy (e.g., append([]byte(nil), data...)).
func (s *OutputStorage) Append(data []byte) {
	if s == nil {
		return
	}

	logger.Printf("Push data: %s\n", string(data))

	newTail := &node{data: data}

	s.tail.next.Store(newTail)
	s.tail = newTail

	logger.Printf("Pushed data: %s\n", string(data))
	s.broadcaster.Publish(struct{}{})

	logger.Printf("Published data: %s\n", string(data))
}

func (s *OutputStorage) subscribeRunningProcess(notifier chan struct{}, ch chan []byte) {
	id := uuid.New()
	logger.Printf("%s Started Running Subscriber:\n", id)
	prev := s.head

	for {
		current := prev.next.Load()
		if current == nil {
			_, ok := <-notifier
			if !ok {
				logger.Printf("%s Notifier closed during send\n", id)
				close(ch)
				return
			}
			// notifier triggered; continue loop to re-check next
			logger.Printf("%s Notifier triggered\n", id)
			continue
		}
		prev = current

		ch <- current.data
		logger.Printf("%s Push data into channel: %s\n", id, string(current.data))
	}
}

func (s *OutputStorage) subscribeStoppedProcess(ch chan []byte) {
	id := uuid.New()
	logger.Printf("%s Started Subscriber:\n", id)
	prev := s.head

	for {
		current := prev.next.Load()
		if current == nil {
			close(ch)
			return
		}
		prev = current

		ch <- current.data
		logger.Printf("%s Push data into channel: %s\n", id, string(current.data))
	}
}

func (s *OutputStorage) Subscribe(capacity int) <-chan []byte {
	ch := make(chan []byte, capacity)
	notifier, err := s.broadcaster.Subscribe()
	if err == nil {
		go s.subscribeRunningProcess(notifier, ch)
	} else {
		go s.subscribeStoppedProcess(ch)
	}

	return ch
}

// ForEach iterates over all stored byte slices in insertion order.
// The iterator function receives each slice; if it returns false, iteration stops early.
func (s *OutputStorage) ForEach(iter func([]byte) bool) {
	if s == nil || iter == nil {
		return
	}
	cur := s.head.next.Load() // skip sentinel
	for cur != nil {
		if !iter(cur.data) {
			return
		}
		cur = cur.next.Load()
	}
}

// Bytes concatenates all stored byte slices into a single slice.
// This is a convenience method and may allocate proportional to total data size.
func (s *OutputStorage) Bytes() []byte {
	// First pass: gather slices and estimate total size.
	total := 0
	slices := make([][]byte, 0, 16)
	s.ForEach(func(b []byte) bool {
		slices = append(slices, b)
		total += len(b)
		return true
	})
	out := make([]byte, 0, total)
	for _, b := range slices {
		out = append(out, b...)
	}
	return out
}

// String returns all stored byte slices concatenated into a single string.
func (s *OutputStorage) String() string {
	return string(s.Bytes())
}
