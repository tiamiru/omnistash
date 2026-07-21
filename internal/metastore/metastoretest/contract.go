package metastoretest

import "testing"

func ExerciseMetadataStoreContract(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	ExerciseAtomicContract(t, newStore)
	ExerciseNamespaceOpsContract(t, newStore)
	ExerciseBlobOpsContract(t, newStore)
	ExerciseManifestOpsContract(t, newStore)
	ExerciseReferrerOpsContract(t, newStore)
}
