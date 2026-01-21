package cancel

import "context"

// ContextCanceler wraps context.Context for cancellation signaling.
//
// This is the standard library approach. Each call to Done() performs
// a select on ctx.Done(), which has overhead from channel operations.
type ContextCanceler struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewContext creates a ContextCanceler from a parent context.
func NewContext(parent context.Context) *ContextCanceler {
	ctx, cancel := context.WithCancel(parent)
	return &ContextCanceler{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Done returns true if the context has been cancelled.
//
// This performs a non-blocking select on ctx.Done().
func (c *ContextCanceler) Done() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

// Cancel triggers cancellation of the context.
func (c *ContextCanceler) Cancel() {
	c.cancel()
}

// Context returns the underlying context.Context.
// Useful for passing to functions that expect a context.
func (c *ContextCanceler) Context() context.Context {
	return c.ctx
}
