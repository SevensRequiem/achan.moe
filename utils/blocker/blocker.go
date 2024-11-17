package blocker

import (
	"sync"

	"github.com/labstack/echo/v4"
)

// Blocker is the struct that holds the block state and synchronization objects
type Blocker struct {
	mu      sync.Mutex
	blocked bool
	cond    *sync.Cond
}

// NewBlocker creates a new Blocker instance
func NewBlocker() *Blocker {
	b := &Blocker{}
	b.cond = sync.NewCond(&b.mu)
	return b
}

// Start blocks all requests until Close is called
func (b *Blocker) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set blocked state to true
	b.blocked = true
}

// Close unblocks all requests and allows them to be processed again
func (b *Blocker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set blocked state to false and signal the condition variable
	b.blocked = false
	b.cond.Broadcast()
}

// Middleware blocks requests when the server is in a "blocked" state
func (b *Blocker) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		b.mu.Lock()
		defer b.mu.Unlock()

		// Wait if blocked, until Close is called
		for b.blocked {
			b.cond.Wait()
		}

		// Continue with the request if not blocked
		return next(c)
	}
}
