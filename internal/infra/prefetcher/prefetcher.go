package prefetcher

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/tasuku43/gion/internal/domain/repo"
)

type Task struct {
	done chan struct{}
	err  error
}

type Prefetcher struct {
	mu      sync.Mutex
	tasks   map[string]*Task
	timeout time.Duration
}

func New(timeout time.Duration) *Prefetcher {
	return &Prefetcher{
		tasks:   make(map[string]*Task),
		timeout: timeout,
	}
}

func Ensure(prefetch *Prefetcher, timeout time.Duration) *Prefetcher {
	if prefetch == nil {
		return New(timeout)
	}
	return prefetch
}

func (p *Prefetcher) Start(ctx context.Context, rootDir, repoSpec string) (bool, error) {
	if p == nil {
		return false, nil
	}
	repoSpec = strings.TrimSpace(repoSpec)
	if repoSpec == "" {
		return false, nil
	}
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		return false, err
	}
	_, exists, err := repo.Exists(rootDir, repoSpec)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	key := strings.TrimSpace(spec.RepoKey)
	if key == "" {
		key = repoSpec
	}

	p.mu.Lock()
	if _, ok := p.tasks[key]; ok {
		p.mu.Unlock()
		return true, nil
	}
	task := &Task{done: make(chan struct{})}
	p.tasks[key] = task
	p.mu.Unlock()

	go func() {
		defer close(task.done)
		fetchCtx := ctx
		cancel := func() {}
		if p.timeout > 0 {
			fetchCtx, cancel = context.WithTimeout(ctx, p.timeout)
		}
		defer cancel()
		if err := repo.Prefetch(fetchCtx, rootDir, repoSpec); err != nil {
			task.err = err
		}
	}()

	return true, nil
}

func (p *Prefetcher) StartAll(ctx context.Context, rootDir string, repoSpecs []string) (bool, error) {
	started := false
	for _, repoSpec := range repoSpecs {
		ok, err := p.Start(ctx, rootDir, repoSpec)
		if err != nil {
			return started, err
		}
		if ok {
			started = true
		}
	}
	return started, nil
}

func (p *Prefetcher) Wait(ctx context.Context, repoSpec string) error {
	if p == nil {
		return nil
	}
	repoSpec = strings.TrimSpace(repoSpec)
	if repoSpec == "" {
		return nil
	}
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		return err
	}
	key := strings.TrimSpace(spec.RepoKey)
	if key == "" {
		key = repoSpec
	}
	p.mu.Lock()
	task := p.tasks[key]
	p.mu.Unlock()
	if task == nil {
		return nil
	}
	select {
	case <-task.done:
		return task.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Prefetcher) WaitAll(ctx context.Context, repoSpecs []string) error {
	for _, repoSpec := range repoSpecs {
		if err := p.Wait(ctx, repoSpec); err != nil {
			return err
		}
	}
	return nil
}
