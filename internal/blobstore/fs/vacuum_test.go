package fs

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/clock/clocktest"
)

func TestFindStaleStagingEntries(t *testing.T) {
	t.Parallel()

	t.Run("happy path: only entries past grace period are returned", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		now := time.Now()
		stale := filepath.Join(dir, "stale")
		fresh := filepath.Join(dir, "fresh")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o600))
		require.NoError(t, os.WriteFile(fresh, []byte("x"), 0o600))
		require.NoError(t, os.Chtimes(stale, now.Add(-time.Hour), now.Add(-time.Hour)))

		entries, statErrs, walkErr := findStaleStagingEntries(t.Context(), dir, time.Minute, now)

		require.NoError(t, walkErr)
		assert.Empty(t, statErrs)
		assert.Equal(t, []string{stale}, entries)
	})

	t.Run("edge case: empty directory returns no entries", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		entries, statErrs, walkErr := findStaleStagingEntries(t.Context(), dir, 0, time.Now())

		require.NoError(t, walkErr)
		assert.Empty(t, statErrs)
		assert.Empty(t, entries)
	})

	t.Run("error path: canceled context aborts the walk", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b"), []byte("x"), 0o600))

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, _, walkErr := findStaleStagingEntries(ctx, dir, 0, time.Now())

		require.ErrorIs(t, walkErr, context.Canceled)
	})

	t.Run("edge case: absent directory returns no entries", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "nonexistent")

		entries, statErrs, walkErr := findStaleStagingEntries(t.Context(), dir, 0, time.Now())

		require.ErrorIs(t, walkErr, os.ErrNotExist)
		assert.Empty(t, statErrs)
		assert.Empty(t, entries)
	})
}

func TestFindStaleStagingEntries_GracePeriodBoundary(t *testing.T) {
	t.Parallel()
	const gracePeriod = time.Minute

	testCases := []struct {
		name      string
		age       time.Duration
		wantStale bool
	}{
		{
			name:      "happy path: well past grace period is stale",
			age:       time.Hour,
			wantStale: true,
		},
		{
			name:      "happy path: freshly written entry is preserved",
			age:       0,
			wantStale: false,
		},
		{
			name:      "edge case: exactly at grace period boundary is stale",
			age:       gracePeriod,
			wantStale: true,
		},
		{
			name:      "edge case: just under grace period boundary is preserved",
			age:       gracePeriod - time.Millisecond,
			wantStale: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			now := time.Now()
			entry := filepath.Join(dir, "entry")
			require.NoError(t, os.WriteFile(entry, []byte("x"), 0o600))
			require.NoError(t, os.Chtimes(entry, now.Add(-tc.age), now.Add(-tc.age)))

			entries, statErrs, walkErr := findStaleStagingEntries(t.Context(), dir, gracePeriod, now)

			require.NoError(t, walkErr)
			assert.Empty(t, statErrs)
			if tc.wantStale {
				assert.Equal(t, []string{entry}, entries)
			} else {
				assert.Empty(t, entries)
			}
		})
	}
}

func TestRemoveCommandPaths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		makePath    func(dir string) string
		wantRemoved int
	}{
		{
			name: "happy path: removes existing file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o600))
			},
			makePath:    func(dir string) string { return filepath.Join(dir, "f") },
			wantRemoved: 1,
		},
		{
			name:        "edge case: absent entry is silently skipped",
			makePath:    func(dir string) string { return filepath.Join(dir, "missing") },
			wantRemoved: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, dir)
			}
			root, err := os.OpenRoot(dir)
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, root.Close()) })

			path := tc.makePath(dir)
			cmd := batchDeleteBlobCommand{root: root, paths: []string{path}}
			result := removeCommandPaths(cmd)

			require.NoError(t, result.errs[0])
			assert.Equal(t, tc.wantRemoved, result.removed)
			_, statErr := os.Stat(path)
			assert.ErrorIs(t, statErr, os.ErrNotExist)
		})
	}
}

func TestVacuumManagerStop(t *testing.T) {
	t.Parallel()
	m := newVacuumManager(slog.Default())
	stop := m.Start()

	err := stop()

	require.NoError(t, err)
}

func TestVacuumManagerDoubleStart(t *testing.T) {
	t.Parallel()
	m := newVacuumManager(slog.Default())
	stop := m.Start()
	t.Cleanup(func() { assert.NoError(t, stop()) })

	nilStop := m.Start()
	require.Nil(t, nilStop)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o600))
	_, _, removeBatchErr := m.removeBatch(t.Context(), dir, []string{filepath.Join(dir, "f")})
	require.NoError(t, removeBatchErr)
}

func TestRemoveBatch_NotStarted(t *testing.T) {
	t.Parallel()
	m := newVacuumManager(slog.Default())
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o600))

	_, _, err := m.removeBatch(t.Context(), dir, []string{filepath.Join(dir, "f")})

	require.ErrorIs(t, err, errVacuumNotStarted)
}

func TestRemoveBatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		makePaths   func(dir string) []string
		wantErr     bool
		wantPathErr bool
		wantRemoved []string
		wantRemain  []string
	}{
		{
			name: "happy path: dispatches every path to a worker and removes all files",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "b"), []byte("x"), 0o600))
			},
			makePaths:   func(dir string) []string { return []string{filepath.Join(dir, "a"), filepath.Join(dir, "b")} },
			wantRemoved: []string{"a", "b"},
		},
		{
			name: "edge case: missing entries in the batch are treated as success",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o600))
			},
			makePaths:   func(dir string) []string { return []string{filepath.Join(dir, "a"), filepath.Join(dir, "missing")} },
			wantRemoved: []string{"a"},
		},
		{
			name: "error path: a genuine removal failure is returned without blocking the rest of the batch",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o600))
				nonEmptyDir := filepath.Join(dir, "nonempty")
				require.NoError(t, os.Mkdir(nonEmptyDir, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(nonEmptyDir, "child"), []byte("x"), 0o600))
			},
			makePaths:   func(dir string) []string { return []string{filepath.Join(dir, "a"), filepath.Join(dir, "nonempty")} },
			wantPathErr: true,
			wantRemoved: []string{"a"},
			wantRemain:  []string{"nonempty"},
		},
		{
			name:        "error path: absolute path outside root returns a path error",
			makePaths:   func(dir string) []string { return []string{"/etc/passwd"} },
			wantPathErr: true,
		},
		{
			name: "error path: path in a sibling directory returns a path error",
			makePaths: func(dir string) []string {
				return []string{filepath.Join(dir, "..", "escape")}
			},
			wantPathErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newVacuumManager(slog.Default())
			stop := m.Start()
			t.Cleanup(func() { assert.NoError(t, stop()) })
			dir := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, dir)
			}

			_, pathErrs, err := m.removeBatch(t.Context(), dir, tc.makePaths(dir))

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			hasPathErr := false
			for _, e := range pathErrs {
				if e != nil {
					hasPathErr = true

					break
				}
			}
			if tc.wantPathErr {
				assert.True(t, hasPathErr, "expected at least one path error")
			} else {
				assert.False(t, hasPathErr, "expected no path errors")
			}
			for _, name := range tc.wantRemoved {
				_, statErr := os.Stat(filepath.Join(dir, name))
				require.ErrorIs(t, statErr, os.ErrNotExist)
			}
			for _, name := range tc.wantRemain {
				_, statErr := os.Stat(filepath.Join(dir, name))
				assert.NoError(t, statErr)
			}
		})
	}
}

func TestVacuumStagingBlobs(t *testing.T) {
	t.Parallel()

	t.Run("happy path: removes stale staging files and preserves fresh ones", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()
		now := time.Now()
		clk := clocktest.NewFake(now)
		s := NewFilesystemBlobStore(baseDir, blobstore.PartitionBlobs, WithClock(clk))
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		stagingDir := filepath.Join(baseDir, string(blobstore.PartitionBlobs), ".staging")
		require.NoError(t, os.MkdirAll(stagingDir, 0o750))
		stale := filepath.Join(stagingDir, "stale")
		fresh := filepath.Join(stagingDir, "fresh")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o600))
		require.NoError(t, os.WriteFile(fresh, []byte("x"), 0o600))
		require.NoError(t, os.Chtimes(stale, now.Add(-time.Hour), now.Add(-time.Hour)))

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.NoError(t, err)
		_, statErr := os.Stat(stale)
		require.ErrorIs(t, statErr, os.ErrNotExist)
		_, statErr = os.Stat(fresh)
		assert.NoError(t, statErr)
	})

	t.Run("edge case: absent staging dir is a no-op", func(t *testing.T) {
		t.Parallel()
		s := NewFilesystemBlobStore(t.TempDir(), blobstore.PartitionBlobs)
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.NoError(t, err)
	})

	t.Run("error path: canceled context returns error immediately", func(t *testing.T) {
		t.Parallel()
		s := NewFilesystemBlobStore(t.TempDir(), blobstore.PartitionBlobs)
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := s.VacuumStagingBlobs(ctx, time.Minute)

		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("error path: vacuum not started returns ErrVacuumNotStarted", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()
		now := time.Now()
		stagingDir := filepath.Join(baseDir, string(blobstore.PartitionBlobs), ".staging")
		require.NoError(t, os.MkdirAll(stagingDir, 0o750))
		stale := filepath.Join(stagingDir, "stale")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o600))
		require.NoError(t, os.Chtimes(stale, now.Add(-time.Hour), now.Add(-time.Hour)))
		s := NewFilesystemBlobStore(baseDir, blobstore.PartitionBlobs)

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.ErrorIs(t, err, errVacuumNotStarted)
	})
}
