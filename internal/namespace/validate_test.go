package namespace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/ocierror"
)

func TestValidateName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "error path: empty", input: "", wantErr: ocierror.ErrNameInvalid},
		{name: "error path: uppercase letter", input: "MyRepo", wantErr: ocierror.ErrNameInvalid},
		{name: "error path: space", input: "my repo", wantErr: ocierror.ErrNameInvalid},
		{name: "error path: leading hyphen", input: "-myrepo", wantErr: ocierror.ErrNameInvalid},
		{name: "error path: trailing hyphen", input: "myrepo-", wantErr: ocierror.ErrNameInvalid},
		{name: "error path: leading slash", input: "/myrepo", wantErr: ocierror.ErrNameInvalid},
		{name: "error path: trailing slash", input: "myrepo/", wantErr: ocierror.ErrNameInvalid},
		{name: "error path: invalid character", input: "my@repo", wantErr: ocierror.ErrNameInvalid},
		{name: "error path: triple underscore", input: "my___repo", wantErr: ocierror.ErrNameInvalid},
		{name: "happy path: simple name", input: "myrepo"},
		{name: "happy path: hyphen separator", input: "my-repo"},
		{name: "happy path: multiple hyphens", input: "my--repo"},
		{name: "happy path: dot separator", input: "my.repo"},
		{name: "happy path: single underscore", input: "my_repo"},
		{name: "happy path: double underscore", input: "my__repo"},
		{name: "happy path: path with slash", input: "myorg/myrepo"},
		{name: "happy path: multi-component path", input: "a/b/c"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateName(tc.input)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			assert.NoError(t, err)
		})
	}
}
