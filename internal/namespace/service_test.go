package namespace_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/namespace"
	mockmeta "github.com/tiamiru/omnistash/internal/testutil/mock"
)

var errUnexpectedStore = errors.New("unexpected store error")

func TestService_CreateNamespace(t *testing.T) {
	t.Parallel()

	fixedTime := time.Unix(1_000_000_000, 0).UTC()

	testCases := []struct {
		name       string
		namespace  string
		setup      func(ms *mockmeta.MetadataStore)
		wantErr    error
		wantResult *namespace.Namespace
	}{
		{
			name:      "error path: empty name",
			namespace: "",
			wantErr:   namespace.ErrNameInvalid,
		},
		{
			name:      "error path: name with uppercase",
			namespace: "MyRepo",
			wantErr:   namespace.ErrNameInvalid,
		},
		{
			name:      "error path: store returns error",
			namespace: "myrepo",
			setup: func(ms *mockmeta.MetadataStore) {
				ms.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				ms.Tx.On("CreateNamespace", mock.Anything, "myrepo").
					Return(metastore.NamespaceRow{}, errUnexpectedStore)
			},
			wantErr: errUnexpectedStore,
		},
		{
			name:      "error path: namespace already exists",
			namespace: "myrepo",
			setup: func(ms *mockmeta.MetadataStore) {
				ms.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				ms.Tx.On("CreateNamespace", mock.Anything, "myrepo").
					Return(metastore.NamespaceRow{}, metastore.ErrNameExists)
			},
			wantErr: namespace.ErrNameExists,
		},
		{
			name:      "happy path: creates namespace",
			namespace: "myrepo",
			setup: func(ms *mockmeta.MetadataStore) {
				ms.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				ms.Tx.On("CreateNamespace", mock.Anything, "myrepo").Return(
					metastore.NamespaceRow{Name: "myrepo", CreatedAt: fixedTime, UpdatedAt: fixedTime},
					nil,
				)
			},
			wantResult: &namespace.Namespace{Name: "myrepo", CreatedAt: fixedTime, UpdatedAt: fixedTime},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ms := &mockmeta.MetadataStore{Tx: &mockmeta.TxOps{}}
			t.Cleanup(func() {
				ms.AssertExpectations(t)
				ms.Tx.AssertExpectations(t)
			})

			if tc.setup != nil {
				tc.setup(ms)
			}

			svc := namespace.NewService(ms)
			ns, err := svc.CreateNamespace(t.Context(), tc.namespace)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)

			if tc.wantResult != nil {
				assert.Equal(t, *tc.wantResult, ns)
			}
		})
	}
}

func TestService_DeleteNamespace(t *testing.T) {
	t.Parallel()

	fixedTime := time.Unix(1_000_000_000, 0).UTC()

	testCases := []struct {
		name      string
		namespace string
		setup     func(ms *mockmeta.MetadataStore)
		wantErr   error
	}{
		{
			name:      "error path: empty name",
			namespace: "",
			wantErr:   namespace.ErrNameInvalid,
		},
		{
			name:      "error path: name with spaces",
			namespace: "my repo",
			wantErr:   namespace.ErrNameInvalid,
		},
		{
			name:      "error path: store returns error",
			namespace: "myrepo",
			setup: func(ms *mockmeta.MetadataStore) {
				ms.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				ms.Tx.On("DeleteNamespace", mock.Anything, "myrepo").
					Return(metastore.NamespaceRow{}, errUnexpectedStore)
			},
			wantErr: errUnexpectedStore,
		},
		{
			name:      "error path: namespace does not exist",
			namespace: "myrepo",
			setup: func(ms *mockmeta.MetadataStore) {
				ms.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				ms.Tx.On("DeleteNamespace", mock.Anything, "myrepo").
					Return(metastore.NamespaceRow{}, metastore.ErrNameUnknown)
			},
			wantErr: namespace.ErrNameUnknown,
		},
		{
			name:      "happy path: deletes namespace",
			namespace: "myrepo",
			setup: func(ms *mockmeta.MetadataStore) {
				ms.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				ms.Tx.On("DeleteNamespace", mock.Anything, "myrepo").Return(
					metastore.NamespaceRow{Name: "myrepo", CreatedAt: fixedTime, UpdatedAt: fixedTime},
					nil,
				)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ms := &mockmeta.MetadataStore{Tx: &mockmeta.TxOps{}}
			t.Cleanup(func() {
				ms.AssertExpectations(t)
				ms.Tx.AssertExpectations(t)
			})

			if tc.setup != nil {
				tc.setup(ms)
			}

			svc := namespace.NewService(ms)
			_, err := svc.DeleteNamespace(t.Context(), tc.namespace)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			assert.NoError(t, err)
		})
	}
}
