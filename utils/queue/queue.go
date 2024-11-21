package queue

import (
	"fmt"
	"sync"
)

// Queue represents a FIFO queue system that holds functions to be executed.
type Queue struct {
	queues map[string]chan func() // A map of named queues, each holding functions.
	wg     sync.WaitGroup         // To wait for all functions to be executed.
	mu     sync.Mutex             // Mutex to protect the queues.
	closed bool                   // Flag to indicate whether the queue system is closed.
}

var Q *Queue

// init initializes the global Q variable and creates the queues.
func init() {
	fmt.Println("Creating Queues...")
	Q = NewQueue() // Initialize the global Q variable
	Q.CreateQueue("thread:create", 500)
	Q.CreateQueue("thread:delete", 10)

	Q.CreateQueue("post:create", 1000)
	Q.CreateQueue("post:delete", 100)

	Q.CreateQueue("mail:send", 100)
	Q.CreateQueue("mail:remind", 100)
	fmt.Println("Queues Created")
}

// NewQueue creates and returns a new FIFO Queue system.
func NewQueue() *Queue {
	return &Queue{
		queues: make(map[string]chan func()),
	}
}

// CreateQueue creates a new named queue with a specified size.
func (q *Queue) CreateQueue(name string, size int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue system is closed")
	}

	// If the queue already exists, return an error.
	if _, exists := q.queues[name]; exists {
		return fmt.Errorf("queue with name %s already exists", name)
	}

	// Create a new channel for the named queue.
	q.queues[name] = make(chan func(), size)
	fmt.Printf("Created queue: %s with size: %d\n", name, size)

	// Start a goroutine to process functions from the queue.
	go q.processQueue(name)

	return nil
}

// processQueue continuously processes functions from the specified queue.
func (q *Queue) processQueue(name string) {
	for fn := range q.queues[name] {
		if fn == nil {
			fmt.Printf("Encountered nil function in queue: %s\n", name)
			continue
		}

		q.wg.Add(1)
		fmt.Printf("Dequeued function from queue: %s\n", name)

		// Run the function in a separate goroutine.
		go func(fn func()) {
			defer q.wg.Done()
			fmt.Printf("Executing function from queue: %s\n", name)
			fn()
			fmt.Printf("Executed function from queue: %s\n", name)
		}(fn)
	}
}

// Enqueue adds a function to a specific named queue.
func (q *Queue) Enqueue(name string, fn func()) error {
	if fn == nil {
		return fmt.Errorf("cannot enqueue nil function to %s", name)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue system is closed, cannot enqueue to %s", name)
	}

	// Find the named queue and enqueue the function.
	queue, exists := q.queues[name]
	if !exists {
		return fmt.Errorf("queue with name %s does not exist", name)
	}

	// Add the function to the queue.
	queue <- fn
	fmt.Printf("Enqueued function to queue: %s\n", name)
	return nil
}

// Close gracefully shuts down the entire queue system, closing all channels.
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return
	}
	q.closed = true

	// Close all queues' channels.
	for name, queue := range q.queues {
		close(queue)
		fmt.Printf("Closed queue: %s\n", name)
	}
}

// Wait blocks until all enqueued functions are executed.
func (q *Queue) Wait() {
	q.wg.Wait()
	fmt.Println("All functions have been executed.")
}
