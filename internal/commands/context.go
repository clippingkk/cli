package commands

import (
	"context"
	"sync"
)

var (
	globalContext context.Context
	contextMutex  sync.RWMutex
)

// SetContext sets the global context for commands
func SetContext(ctx context.Context) {
	contextMutex.Lock()
	defer contextMutex.Unlock()
	globalContext = ctx
}

// GetContext returns the global context
func GetContext() context.Context {
	contextMutex.RLock()
	defer contextMutex.RUnlock()
	if globalContext == nil {
		return context.Background()
	}
	return globalContext
}