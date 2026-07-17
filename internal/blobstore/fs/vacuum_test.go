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

	"github.com/tiamiru/omnistash/internal/clock/clocktest"
)

const (
	testNamespace  = "library/test"
	otherNamespace = "library/other"
)

func TestFindStaleStagingEntries(t *testing.T) {
	t.Parallel()
	now := time.Now()
	const gracePeriod = time.Minute

	testCases := []struct {
		name             string
		setup            func(t *testing.T, dir string)
		absentDir        bool
		cancelCtx        bool
		gracePeriod      time.Duration
		wantStaleEntries []string
		wantWalkErr      error
	}{
		{
			name: "error path: canceled context aborts the walk",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "b"), []byte("x"), 0o600))
			},
			cancelCtx:   true,
			gracePeriod: 0,
			wantWalkErr: context.Canceled,
		},
		{
			name:        "edge case: absent directory returns walk error",
			absentDir:   true,
			gracePeriod: 0,
			wantWalkErr: os.ErrNotExist,
		},
		{
			name:        "edge case: empty directory returns no entries",
			gracePeriod: 0,
		},
		{
			name: "edge case: exactly at grace period boundary is stale",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				f := filepath.Join(dir, "entry")
				require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))
				require.NoError(t, os.Chtimes(f, now.Add(-gracePeriod), now.Add(-gracePeriod)))
			},
			gracePeriod:      gracePeriod,
			wantStaleEntries: []string{"entry"},
		},
		{
			name: "edge case: just under grace period boundary is preserved",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				f := filepath.Join(dir, "entry")
				require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))
				modTime := now.Add(-(gracePeriod - time.Millisecond))
				require.NoError(t, os.Chtimes(f, modTime, modTime))
			},
			gracePeriod: gracePeriod,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tc.absentDir {
				dir = filepath.Join(dir, "nonexistent")
			}
			if tc.setup != nil {
				tc.setup(t, dir)
			}
			ctx := t.Context()
			if tc.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			entries, statErrs, walkErr := findStaleStagingEntries(ctx, dir, tc.gracePeriod, now)

			require.ErrorIs(t, walkErr, tc.wantWalkErr)
			assert.Empty(t, statErrs)
			var wantEntries []string
			for _, name := range tc.wantStaleEntries {
				wantEntries = append(wantEntries, filepath.Join(dir, name))
			}
			assert.Equal(t, wantEntries, entries)
		})
	}
}

func TestRemoveCommandPaths(t *testing.T) {
	t.Parallel()

	makeRoot := func(t *testing.T) (*os.Root, string) {
		t.Helper()
		dir := t.TempDir()
		root, err := os.OpenRoot(dir)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, root.Close()) })

		return root, dir
	}

	testCases := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		paths       []string // absolute paths used as-is; relative paths joined with dir
		wantErr     error
		wantRemoved int
		wantGone    []string
	}{
		{
			name:    "error path: absolute path outside root returns path error",
			paths:   []string{"/etc/passwd"},
			wantErr: errPathEscapesRoot,
		},
		{
			name:    "error path: path in sibling directory returns path error",
			paths:   []string{"../escape"},
			wantErr: errPathEscapesRoot,
		},
		{
			name:    "error path: path equal to root dir returns path error",
			paths:   []string{"."},
			wantErr: errPathEscapesRoot,
		},
		{
			name:  "edge case: absent entry is silently skipped",
			paths: []string{"missing"},
		},
		{
			name: "happy path: removes existing file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o600))
			},
			paths:       []string{"f"},
			wantRemoved: 1,
			wantGone:    []string{"f"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, dir := makeRoot(t)
			if tc.setup != nil {
				tc.setup(t, dir)
			}

			paths := make([]string, len(tc.paths))
			for i, p := range tc.paths {
				if filepath.IsAbs(p) {
					paths[i] = p
				} else {
					paths[i] = filepath.Join(dir, p)
				}
			}
			result := removeCommandPaths(batchDeleteBlobCommand{root: root, paths: paths})

			if tc.wantErr != nil {
				require.ErrorIs(t, result.errs[0], tc.wantErr)

				return
			}
			require.NoError(t, result.errs[0])
			assert.Equal(t, tc.wantRemoved, result.removed)
			for _, name := range tc.wantGone {
				_, statErr := os.Stat(filepath.Join(dir, name))
				assert.ErrorIs(t, statErr, os.ErrNotExist)
			}
		})
	}
}

func TestVacuumManager(t *testing.T) {
	t.Parallel()

	t.Run("edge case: second Start returns nil while first is running", func(t *testing.T) {
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
	})

	t.Run("edge case: stop is idempotent", func(t *testing.T) {
		t.Parallel()
		m := newVacuumManager(slog.Default())
		stop := m.Start()

		err1 := stop()
		err2 := stop()

		require.NoError(t, err1)
		require.NoError(t, err2)
	})
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
			name:      "edge case: empty paths list is a no-op",
			makePaths: func(dir string) []string { return []string{} },
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
			name: "happy path: dispatches every path to a worker and removes all files",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "b"), []byte("x"), 0o600))
			},
			makePaths:   func(dir string) []string { return []string{filepath.Join(dir, "a"), filepath.Join(dir, "b")} },
			wantRemoved: []string{"a", "b"},
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

	t.Run("error path: canceled context returns error immediately", func(t *testing.T) {
		t.Parallel()
		s := NewFilesystemBlobStore(t.TempDir())
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
		stagingDir := filepath.Join(baseDir, testNamespace, ".staging")
		require.NoError(t, os.MkdirAll(stagingDir, 0o750))
		stale := filepath.Join(stagingDir, "stale")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o600))
		require.NoError(t, os.Chtimes(stale, now.Add(-time.Hour), now.Add(-time.Hour)))
		s := NewFilesystemBlobStore(baseDir)

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.ErrorIs(t, err, errVacuumNotStarted)
	})

	t.Run("error path: unreadable staging dir propagates walk error", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()
		stagingDir := filepath.Join(baseDir, testNamespace, ".staging")
		require.NoError(t, os.MkdirAll(stagingDir, 0o750))
		require.NoError(t, os.Chmod(stagingDir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(stagingDir, 0o600) })
		s := NewFilesystemBlobStore(baseDir)
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("edge case: absent prefix dir is a no-op", func(t *testing.T) {
		t.Parallel()
		s := NewFilesystemBlobStore(t.TempDir())
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.NoError(t, err)
	})

	t.Run("happy path: removes stale staging files and preserves fresh ones", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()
		now := time.Now()
		clk := clocktest.NewFake(now)
		s := NewFilesystemBlobStore(baseDir, WithClock(clk))
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		stagingDir := filepath.Join(baseDir, testNamespace, ".staging")
		require.NoError(t, os.MkdirAll(stagingDir, 0o750))
		stale := filepath.Join(stagingDir, "stale")
		fresh := filepath.Join(stagingDir, "fresh")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o600))
		require.NoError(t, os.WriteFile(fresh, []byte("x"), 0o600))
		require.NoError(t, os.Chtimes(stale, now.Add(-time.Hour), now.Add(-time.Hour)))
		require.NoError(t, os.Chtimes(fresh, now, now))

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.NoError(t, err)
		_, statErr := os.Stat(stale)
		require.ErrorIs(t, statErr, os.ErrNotExist)
		_, statErr = os.Stat(fresh)
		assert.NoError(t, statErr)
	})

	t.Run("happy path: removes stale files across multiple namespaces", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()
		now := time.Now()
		clk := clocktest.NewFake(now)
		s := NewFilesystemBlobStore(baseDir, WithClock(clk))
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		staleFiles := make([]string, 0, 2)
		for _, ns := range []string{otherNamespace, testNamespace} {
			stagingDir := filepath.Join(baseDir, ns, ".staging")
			require.NoError(t, os.MkdirAll(stagingDir, 0o750))
			stale := filepath.Join(stagingDir, "stale")
			require.NoError(t, os.WriteFile(stale, []byte("x"), 0o600))
			require.NoError(t, os.Chtimes(stale, now.Add(-time.Hour), now.Add(-time.Hour)))
			staleFiles = append(staleFiles, stale)
		}

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.NoError(t, err)
		for _, stale := range staleFiles {
			_, statErr := os.Stat(stale)
			assert.ErrorIs(t, statErr, os.ErrNotExist)
		}
	})

	t.Run("happy path: stale files in one namespace do not affect fresh files in another", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()
		now := time.Now()
		clk := clocktest.NewFake(now)
		s := NewFilesystemBlobStore(baseDir, WithClock(clk))
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		// otherNamespace has a stale file.
		staleStagingDir := filepath.Join(baseDir, otherNamespace, ".staging")
		require.NoError(t, os.MkdirAll(staleStagingDir, 0o750))
		stale := filepath.Join(staleStagingDir, "stale")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o600))
		require.NoError(t, os.Chtimes(stale, now.Add(-time.Hour), now.Add(-time.Hour)))

		// testNamespace has a fresh file.
		freshStagingDir := filepath.Join(baseDir, testNamespace, ".staging")
		require.NoError(t, os.MkdirAll(freshStagingDir, 0o750))
		fresh := filepath.Join(freshStagingDir, "fresh")
		require.NoError(t, os.WriteFile(fresh, []byte("x"), 0o600))
		require.NoError(t, os.Chtimes(fresh, now, now))

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.NoError(t, err)
		_, statErr := os.Stat(stale)
		require.ErrorIs(t, statErr, os.ErrNotExist)
		_, statErr = os.Stat(fresh)
		assert.NoError(t, statErr)
	})

	t.Run("error path: unreadable staging dir in one namespace does not block cleanup of another", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()
		now := time.Now()
		clk := clocktest.NewFake(now)
		s := NewFilesystemBlobStore(baseDir, WithClock(clk))
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		// otherNamespace staging dir is unreadable (walked first, lexicographically).
		badStagingDir := filepath.Join(baseDir, otherNamespace, ".staging")
		require.NoError(t, os.MkdirAll(badStagingDir, 0o750))
		require.NoError(t, os.Chmod(badStagingDir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(badStagingDir, 0o600) })

		// testNamespace has a stale file that should still be removed (walked second).
		goodStagingDir := filepath.Join(baseDir, testNamespace, ".staging")
		require.NoError(t, os.MkdirAll(goodStagingDir, 0o750))
		stale := filepath.Join(goodStagingDir, "stale")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o600))
		require.NoError(t, os.Chtimes(stale, now.Add(-time.Hour), now.Add(-time.Hour)))

		err := s.VacuumStagingBlobs(t.Context(), time.Minute)

		require.ErrorIs(t, err, os.ErrPermission)
		_, statErr := os.Stat(stale)
		assert.ErrorIs(t, statErr, os.ErrNotExist)
	})
}
