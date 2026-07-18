package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRangeHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		header    string
		wantErr   error
		wantFirst int64
		wantLast  int64
	}{
		{
			name:    "error path: missing bytes= prefix",
			header:  "0-499",
			wantErr: errRangeHeaderInvalid,
		},
		{
			name:    "error path: open-ended range",
			header:  "bytes=0-",
			wantErr: errRangeHeaderInvalid,
		},
		{
			name:    "error path: suffix range",
			header:  "bytes=-500",
			wantErr: errRangeHeaderInvalid,
		},
		{
			name:    "error path: empty string",
			header:  "",
			wantErr: errRangeHeaderInvalid,
		},
		{
			name:    "error path: first greater than last",
			header:  "bytes=500-0",
			wantErr: errRangeHeaderInvalid,
		},
		{
			name:      "happy path: zero-based range",
			header:    "bytes=0-499",
			wantFirst: 0,
			wantLast:  499,
		},
		{
			name:      "happy path: single byte",
			header:    "bytes=100-100",
			wantFirst: 100,
			wantLast:  100,
		},
		{
			name:      "happy path: large offsets",
			header:    "bytes=1048576-2097151",
			wantFirst: 1048576,
			wantLast:  2097151,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			first, last, err := parseRangeHeader(tc.header)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantFirst, first)
			assert.Equal(t, tc.wantLast, last)
		})
	}
}

func TestParseContentRangeHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		header    string
		wantErr   error
		wantFirst int64
		wantLast  int64
	}{
		{
			name:    "error path: bytes= prefix not accepted",
			header:  "bytes=0-499",
			wantErr: errContentRangeHeaderInvalid,
		},
		{
			name:    "error path: open-ended range",
			header:  "0-",
			wantErr: errContentRangeHeaderInvalid,
		},
		{
			name:    "error path: empty string",
			header:  "",
			wantErr: errContentRangeHeaderInvalid,
		},
		{
			name:    "error path: first greater than last",
			header:  "500-0",
			wantErr: errContentRangeHeaderInvalid,
		},
		{
			name:      "happy path: zero-based range",
			header:    "0-499",
			wantFirst: 0,
			wantLast:  499,
		},
		{
			name:      "happy path: single byte",
			header:    "0-0",
			wantFirst: 0,
			wantLast:  0,
		},
		{
			name:      "happy path: large offsets",
			header:    "1048576-2097151",
			wantFirst: 1048576,
			wantLast:  2097151,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			first, last, err := parseContentRangeHeader(tc.header)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantFirst, first)
			assert.Equal(t, tc.wantLast, last)
		})
	}
}
