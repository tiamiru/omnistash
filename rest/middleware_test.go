package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitNamespacedPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		urlPath  string
		wantOK   bool
		wantName string
		wantTail string
	}{
		{
			name:    "error path: missing /v2/ prefix",
			urlPath: "/v1/myrepo/blobs/sha256:abc",
			wantOK:  false,
		},
		{
			name:    "error path: empty string",
			urlPath: "",
			wantOK:  false,
		},
		{
			name:    "error path: no OCI separator",
			urlPath: "/v2/myrepo/somethingelse",
			wantOK:  false,
		},
		{
			name:    "error path: empty name",
			urlPath: "/v2//blobs/sha256:abc",
			wantOK:  false,
		},
		{
			name:     "happy path: blobs separator",
			urlPath:  "/v2/myrepo/blobs/sha256:abc",
			wantOK:   true,
			wantName: "myrepo",
			wantTail: "/blobs/sha256:abc",
		},
		{
			name:     "happy path: manifests separator",
			urlPath:  "/v2/myrepo/manifests/latest",
			wantOK:   true,
			wantName: "myrepo",
			wantTail: "/manifests/latest",
		},
		{
			name:     "happy path: tags separator",
			urlPath:  "/v2/myrepo/tags/list",
			wantOK:   true,
			wantName: "myrepo",
			wantTail: "/tags/list",
		},
		{
			name:     "happy path: referrers separator",
			urlPath:  "/v2/myrepo/referrers/sha256:abc",
			wantOK:   true,
			wantName: "myrepo",
			wantTail: "/referrers/sha256:abc",
		},
		{
			name:     "happy path: multi-component name",
			urlPath:  "/v2/myorg/myrepo/blobs/sha256:abc",
			wantOK:   true,
			wantName: "myorg/myrepo",
			wantTail: "/blobs/sha256:abc",
		},
		{
			name:     "happy path: picks rightmost separator",
			urlPath:  "/v2/myrepo/blobs/sha256:abc/blobs/sha256:def",
			wantOK:   true,
			wantName: "myrepo/blobs/sha256:abc",
			wantTail: "/blobs/sha256:def",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			name, tail, ok := splitNamespacedPath(tc.urlPath)

			if !tc.wantOK {
				assert.False(t, ok)

				return
			}

			assert.True(t, ok)
			assert.Equal(t, tc.wantName, name)
			assert.Equal(t, tc.wantTail, tail)
		})
	}
}
