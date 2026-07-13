package health_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tiamiru/omnistash/internal/health"
)

func TestChecker_Check(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		version string
		commit  string
		date    string
		want    health.Status
	}{
		{
			name:    "happy path: returns ok with provided version",
			version: "1.2.3",
			commit:  "abc1234",
			date:    "2026-07-14T00:00:00Z",
			want:    health.Status{Status: "ok", Version: "1.2.3", Commit: "abc1234", Date: "2026-07-14T00:00:00Z"},
		},
		{
			name:    "happy path: returns ok with dev version",
			version: "dev",
			commit:  "none",
			date:    "unknown",
			want:    health.Status{Status: "ok", Version: "dev", Commit: "none", Date: "unknown"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			checker := health.NewChecker(tc.version, tc.commit, tc.date)
			got := checker.Check()

			assert.Equal(t, tc.want, got)
		})
	}
}
