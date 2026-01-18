package cli

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/tasuku43/gwst/internal/domain/repo"
)

type prefetchTask struct {
	done chan struct{}
	err  error
}

type prefetcher struct {
	mu      sync.Mutex
	tasks   map[string]*prefetchTask
	timeout time.Duration
}

func newPrefetcher(timeout time.Duration) *prefetcher {
	return &prefetcher{
		tasks:   make(map[string]*prefetchTask),
		timeout: timeout,
	}
}

func (p *prefetcher) start(ctx context.Context, rootDir, repoSpec string) (bool, error) {
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
	task := &prefetchTask{done: make(chan struct{})}
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

func (p *prefetcher) startAll(ctx context.Context, rootDir string, repoSpecs []string) (bool, error) {
	started := false
	for _, repoSpec := range repoSpecs {
		ok, err := p.start(ctx, rootDir, repoSpec)
		if err != nil {
			return started, err
		}
		if ok {
			started = true
		}
	}
	return started, nil
}

func (p *prefetcher) wait(ctx context.Context, repoSpec string) error {
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

func (p *prefetcher) waitAll(ctx context.Context, repoSpecs []string) error {
	for _, repoSpec := range repoSpecs {
		if err := p.wait(ctx, repoSpec); err != nil {
			return err
		}
	}
	return nil
}

func ensurePrefetcher(prefetch *prefetcher) *prefetcher {
	if prefetch == nil {
		return newPrefetcher(defaultPrefetchTimeout)
	}
	return prefetch
}
