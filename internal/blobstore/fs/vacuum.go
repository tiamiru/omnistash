package fs

import (
	"context"
	"errors"
	"fmt"
	iofs "io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tiamiru/omnistash/internal/logtag"
)

var (
	errVacuumNotStarted = errors.New("vacuum not started")
	errPathEscapesRoot  = errors.New("path escapes root")
)

const defaultVacuumWorkerPoolSize = 3

type batchDeleteResult struct {
	removed int
	errs    []error
}

type batchDeleteBlobCommand struct {
	root  *os.Root
	paths []string
	done  chan<- batchDeleteResult
}

type VacuumManager struct {
	mu       sync.RWMutex
	jobs     chan batchDeleteBlobCommand
	poolSize int
	logger   *slog.Logger
}

func newVacuumManager(logger *slog.Logger) *VacuumManager {
	return &VacuumManager{
		poolSize: defaultVacuumWorkerPoolSize,
		logger:   logger,
	}
}

// Start launches the worker pool and returns a stop function.
// Returns nil if the worker pool already started.
func (m *VacuumManager) Start() func() error {
	m.mu.Lock()
	if m.jobs != nil {
		m.mu.Unlock()

		return nil
	}
	jobs := make(chan batchDeleteBlobCommand, m.poolSize)
	m.jobs = jobs
	m.mu.Unlock()

	var wg sync.WaitGroup
	for range m.poolSize {
		wg.Go(func() {
			for cmd := range jobs {
				cmd.done <- removeCommandPaths(cmd)
			}
		})
	}

	var once sync.Once

	return func() error {
		once.Do(func() {
			m.mu.Lock()
			if m.jobs == jobs {
				m.jobs = nil
				close(jobs)
			}
			m.mu.Unlock()

			wg.Wait()
		})

		return nil
	}
}

func (m *VacuumManager) removeBatch(ctx context.Context, dir string, paths []string) (int, []error, error) {
	m.mu.RLock()
	jobs := m.jobs
	if jobs == nil {
		m.mu.RUnlock()

		return 0, nil, fmt.Errorf("removeBatch: %w", errVacuumNotStarted)
	}
	defer m.mu.RUnlock()

	if len(paths) == 0 {
		return 0, nil, nil
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return 0, make([]error, len(paths)), nil
		}

		return 0, nil, fmt.Errorf("removeBatch: open root: %w", err)
	}
	defer func() {
		closeErr := root.Close()
		if closeErr != nil {
			m.logger.Warn("removeBatch: close root", logtag.Err(closeErr))
		}
	}()

	done := make(chan batchDeleteResult, 1)
	select {
	case jobs <- batchDeleteBlobCommand{root: root, paths: paths, done: done}:
	case <-ctx.Done():
		return 0, nil, fmt.Errorf("removeBatch: %w", ctx.Err())
	}

	result := <-done

	return result.removed, result.errs, nil
}

func (s *FilesystemBlobStore) VacuumStagingBlobs(
	ctx context.Context,
	gracePeriod time.Duration,
) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("vacuum staging blobs: %w", err)
	}

	stagingDir := filepath.Join(s.prefix, string(s.partition), ".staging")
	entries, statErrs, walkErr := findStaleStagingEntries(ctx, stagingDir, gracePeriod, s.clock.Now())

	for _, statErr := range statErrs {
		s.logger.Warn("VacuumStagingBlobs: stat entry", logtag.Err(statErr))
	}

	var errs []error
	if len(entries) > 0 {
		_, pathErrs, removeErr := s.vacuumManager.removeBatch(ctx, stagingDir, entries)
		if removeErr != nil {
			errs = append(errs, removeErr)
		}
		errs = append(errs, pathErrs...)
	}

	if walkErr != nil && !errors.Is(walkErr, iofs.ErrNotExist) {
		errs = append(errs, fmt.Errorf("vacuum staging blobs: walk: %w", walkErr))
	}

	return errors.Join(errs...)
}

func findStaleStagingEntries(
	ctx context.Context,
	dir string,
	gracePeriod time.Duration,
	now time.Time,
) ([]string, []error, error) {
	var entries []string
	var statErrs []error
	walkErr := filepath.WalkDir(dir, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			if path == dir {
				return err
			}
			statErrs = append(statErrs, fmt.Errorf("vacuum staging blobs: walk %s: %w", path, err))

			return nil
		}
		if d.IsDir() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			statErrs = append(statErrs, fmt.Errorf("vacuum staging blobs: stat %s: %w", path, infoErr))

			return nil
		}
		if now.Sub(info.ModTime()) < gracePeriod {
			return nil
		}

		entries = append(entries, path)

		return nil
	})

	return entries, statErrs, walkErr
}

func removeCommandPaths(cmd batchDeleteBlobCommand) batchDeleteResult {
	removed := 0
	errs := make([]error, len(cmd.paths))
	for i, path := range cmd.paths {
		rel, relErr := filepath.Rel(cmd.root.Name(), path)
		if relErr != nil || rel == "." || strings.HasPrefix(rel, "..") {
			errs[i] = fmt.Errorf("remove %s: %w", path, errPathEscapesRoot)

			continue
		}
		removeErr := cmd.root.Remove(rel)
		if removeErr == nil {
			removed++
		} else if !errors.Is(removeErr, iofs.ErrNotExist) {
			errs[i] = fmt.Errorf("remove %s: %w", path, removeErr)
		}
	}

	return batchDeleteResult{removed: removed, errs: errs}
}
