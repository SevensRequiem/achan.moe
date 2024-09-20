package queue

import (
	"fmt"
	"sync"
)

// Queue represents a thread-safe FIFO queue to process functions.
type Queue struct {
	Name  string
	mu    sync.Mutex
	cond  *sync.Cond
	queue chan func()
	stop  chan struct{}
	wg    sync.WaitGroup
}

// New creates a new Queue with a given name and buffer size.
func New(name string, bufferSize int) *Queue {
	q := &Queue{
		Name:  name,
		queue: make(chan func(), bufferSize),
		stop:  make(chan struct{}),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Enqueue adds a function to the queue. Returns error if the queue is full.
func (q *Queue) Enqueue(f func()) error {
	select {
	case q.queue <- f:
		fmt.Printf("Function added to queue %s\n", q.Name)
		return nil
	default:
		return fmt.Errorf("queue %s is full", q.Name)
	}
}

// Dequeue removes and returns the next function from the queue with error handling.
func (q *Queue) Dequeue() (func(), error) {
	select {
	case f := <-q.queue:
		fmt.Printf("Function dequeued from queue %s\n", q.Name)
		return f, nil
	case <-q.stop:
		return nil, fmt.Errorf("queue %s is stopped", q.Name)
	}
}

// IsEmpty checks if the queue is empty.
func (q *Queue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue) == 0
}

// Stop signals the queue to stop processing.
func (q *Queue) Stop() {
	close(q.stop)
	q.wg.Wait()
	fmt.Printf("Queue %s stop signal received\n", q.Name)
}

// Size returns the current size of the queue.
func (q *Queue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue)
}

// Process starts processing the queue in a separate goroutine.
func (q *Queue) Process() {
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case f, ok := <-q.queue:
				if !ok {
					fmt.Printf("Queue %s is closed, stopping processing\n", q.Name)
					return
				}
				fmt.Printf("Function executing from queue %s\n", q.Name)
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("Function panicked from queue %s: %v\n", q.Name, r)
						}
					}()
					f() // Execute the function
				}()
				fmt.Printf("Function executed from queue %s\n", q.Name)
			case <-q.stop:
				fmt.Printf("Queue %s processing stopped\n", q.Name)
				return
			}
		}
	}()
}

// QueueManager manages multiple named queues.
type QueueManager struct {
	queues map[string]*Queue
	mu     sync.Mutex
}

// NewQueueManager creates a new QueueManager.
func NewQueueManager() *QueueManager {
	return &QueueManager{
		queues: make(map[string]*Queue),
	}
}

// GetQueue retrieves a queue by name, creating it if it doesn't exist.
func (qm *QueueManager) GetQueue(name string, bufferSize int) *Queue {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	if q, exists := qm.queues[name]; exists {
		return q
	}
	q := New(name, bufferSize)
	qm.queues[name] = q
	fmt.Printf("Queue %s created\n", name)
	return q
}

// StopAll stops all queues managed by the QueueManager.
func (qm *QueueManager) StopAll() {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	for name, q := range qm.queues {
		fmt.Printf("Stopping queue %s\n", name)
		q.Stop()
	}
}

// ProcessQueuesWithPrefix starts processing queues with the specified prefix.
func (qm *QueueManager) ProcessQueuesWithPrefix(prefix string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	for name, q := range qm.queues {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			fmt.Printf("Starting processing for queue %s\n", name)
			q.Process()
		}
	}
}

// ProcessAll starts processing all queues managed by the QueueManager.
func (qm *QueueManager) ProcessAll() {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	for name, q := range qm.queues {
		fmt.Printf("Starting processing for queue %s\n", name)
		q.Process()
	}
}
