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
		want    health.Status
	}{
		{
			name:    "happy path: returns ok with provided version",
			version: "1.2.3",
			want:    health.Status{Status: "ok", Version: "1.2.3"},
		},
		{
			name:    "happy path: returns ok with dev version",
			version: "dev",
			want:    health.Status{Status: "ok", Version: "dev"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			checker := health.NewChecker(tc.version)
			got := checker.Check()

			assert.Equal(t, tc.want, got)
		})
	}
}
