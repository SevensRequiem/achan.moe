package queue

import (
	"container/list"
	"fmt"
	"sync"
)

// Queue represents a thread-safe FIFO queue to process functions.
type Queue struct {
	mu    sync.Mutex
	cond  *sync.Cond
	queue *list.List
	stop  bool
}

// New creates a new Queue.
func New() *Queue {
	q := &Queue{
		queue: list.New(),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Enqueue adds a function to the queue.
func (q *Queue) Enqueue(f func()) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue.PushBack(f)
	q.cond.Signal() // Signal that an item has been added
	fmt.Println("Function added to queue")
}

// Dequeue removes and returns the next function from the queue.
func (q *Queue) Dequeue() func() {
	q.mu.Lock()
	defer q.mu.Unlock()
	for q.queue.Len() == 0 && !q.stop {
		q.cond.Wait() // Wait until there's an item in the queue or stop signal
	}
	if q.queue.Len() == 0 {
		return nil
	}
	front := q.queue.Front()
	q.queue.Remove(front)
	return front.Value.(func())
}

// IsEmpty checks if the queue is empty.
func (q *Queue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.queue.Len() == 0
}

// Stop signals the queue to stop processing.
func (q *Queue) Stop() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stop = true
	q.cond.Broadcast() // Wake up all waiting goroutines
}

// Process starts processing the queue in a separate goroutine.
func (q *Queue) Process() {
	go func() {
		for {
			f := q.Dequeue()
			if f != nil {
				f()
				fmt.Println("Function executed from queue")
			}
			if q.stop && q.IsEmpty() {
				fmt.Println("Queue processing stopped")
				break
			}
		}
	}()
}
