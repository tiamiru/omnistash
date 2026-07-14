package metastoretest

import "testing"

func ExerciseMetadataStoreContract(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	ExerciseAtomicContract(t, newStore)
	ExerciseTxOpsContract(t, newStore)
}
