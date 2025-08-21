package output_storage

import (
	"fmt"
	"sync"
)

type Broadcaster[T any] struct {
	messageReceiver chan T
	mu              sync.Mutex
	subscribers     map[chan T]struct{}
	stopped         bool
}

func RunNewBroadcaster[T any]() *Broadcaster[T] {
	broadcaster := &Broadcaster[T]{
		messageReceiver: make(chan T, 1),
		subscribers:     make(map[chan T]struct{}),
	}

	go broadcaster.start()

	return broadcaster
}

func (broadcaster *Broadcaster[T]) start() {
	logger.Println("Starting broadcaster")

	messageReceiver := broadcaster.messageReceiver

	for {
		select {
		case msg, ok := <-messageReceiver:
			if !ok {
				logger.Println("Received message, but channel is closed")
				broadcaster.mu.Lock()
				for subscriberSender := range broadcaster.subscribers {
					logger.Println("Closing subscriber channel")
					close(subscriberSender)
				}
				broadcaster.stopped = true
				broadcaster.mu.Unlock()

				logger.Println("Stopping broadcaster")

				return
			}

			logger.Println("Received message")
			// Copy the map to avoid holding the lock for a long time.
			broadcaster.mu.Lock()
			subscribers := make([]chan T, 0, len(broadcaster.subscribers))
			for s := range broadcaster.subscribers {
				subscribers = append(subscribers, s)
			}
			broadcaster.mu.Unlock()

			for _, s := range subscribers {
				//use non-blocking send
				select {
				case s <- msg:
					logger.Println("Sent message without blocking")
				default:
					logger.Println("Channel is full, pulling a value")
					// channel is full, drop the first message
					select {
					case <-s:
					default:
					}
					logger.Println("Pulled a value, pushing a new value")
					s <- msg
				}
			}
		}
	}

}

func (broadcaster *Broadcaster[T]) Stop() {
	close(broadcaster.messageReceiver)
}

func (broadcaster *Broadcaster[T]) Subscribe() (chan T, error) {
	// Use a buffer of 1 so we can drop stale notifications without blocking.
	ch := make(chan T, 1)
	broadcaster.mu.Lock()
	if broadcaster.stopped {
		broadcaster.mu.Unlock()
		logger.Println("Can't subscribe")
		return nil, fmt.Errorf("failed to subscribe: broadcaster is stopped")
	}
	broadcaster.subscribers[ch] = struct{}{}
	broadcaster.mu.Unlock()
	logger.Println("New subscriber")
	return ch, nil
}

func (broadcaster *Broadcaster[T]) Unsubscribe(subscriberSender chan T) {
	broadcaster.mu.Lock()
	delete(broadcaster.subscribers, subscriberSender)
	stopped := broadcaster.stopped
	broadcaster.mu.Unlock()
	if !stopped {
		close(subscriberSender)
	}
	logger.Println("Unsubscribed")
}

func (broadcaster *Broadcaster[T]) Publish(msg T) {
	logger.Printf("PUBLISH: START")
	select {
	case broadcaster.messageReceiver <- msg:
		logger.Println("PUBLISH: Sent message without blocking")
	default:
		logger.Println("PUBLISH: Channel is full, pulling a value")
		// channel is full, drop the first message
		select {
		case <-broadcaster.messageReceiver:
		default:
		}

		logger.Println("PUBLISH: Pulled a value, pushing a new value")
		broadcaster.messageReceiver <- msg
	}

	logger.Printf("PUBLISH: FINISH")
}
