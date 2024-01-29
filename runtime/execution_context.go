package runtime

import (
	"context"
	"sync"

	"github.com/anchore/go-logger"
	"github.com/anchore/stereoscope/internal/log"
)

type TempDirProvider interface {
	NewDirectory(name ...string) (string, error)
	Cleanup() error
}

// ExecutionContext contains access to all the execution context needed by image providers
type ExecutionContext interface {
	TempDirProvider
	Context() context.Context
	RegisterCleanup(func() error)
	Log() logger.Logger
	// Execute(func() error) or Executor().Execute(...)
	// Notify(msg any) or Bus().Notify(...)
}

func NewExecutionContext(ctx context.Context, tempDir TempDirProvider) ExecutionContext {
	return &executionContext{
		ctx:      ctx,
		cleanups: nil,
		tmp:      tempDir,
	}
}

type executionContext struct {
	mu       sync.Mutex
	ctx      context.Context
	cleanups []func() error
	tmp      TempDirProvider
}

func (p *executionContext) Context() context.Context {
	return p.ctx
}

func (p *executionContext) RegisterCleanup(cleanupFunc func() error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanups = append(p.cleanups, cleanupFunc)
}

func (p *executionContext) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, cleanup := range p.cleanups {
		withPanicRecovery(p, cleanup)
	}
	p.cleanups = nil
	return nil
}

func withPanicRecovery(ctx ExecutionContext, fn func() error) {
	defer func() {
		if err := recover(); err != nil {
			ctx.Log().Errorf("recovered from panic due to: %v", err)
		}
	}()
	execute(ctx, fn)
}

func (p *executionContext) NewDirectory(name ...string) (string, error) {
	p.RegisterCleanup(p.tmp.Cleanup)
	return p.tmp.NewDirectory(name...)
}

func (p *executionContext) TempDirProvider() TempDirProvider {
	return p.tmp
}

func (p *executionContext) Log() logger.Logger {
	return log.Log
}

func execute(ctx ExecutionContext, fn func() error) {
	err := fn()
	if err != nil {
		ctx.Log().Errorf("error executing function: %w", err)
	}
}

var _ ExecutionContext = (*executionContext)(nil)
