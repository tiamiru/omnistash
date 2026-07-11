package blobstoretest

import (
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
	"github.com/tiamiru/omnistash/internal/blobstore"
)

const (
	DefaultPartition = "default"

	TestContent  = "{}"
	OtherContent = "other"

	MalformedDigest = "notadigest"
	FakeUploadID    = "00000000-0000-4000-8000-000000000000"

	ZeroDigest  digest.Digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	TestDigest  digest.Digest = "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	OtherDigest digest.Digest = "sha256:d9298a10d1b0735837dc4bd85dac641b0f3cef27a47e5d53a54f2f3f5b2fcffa"

	TestDigest512 digest.Digest = "sha512:" +
		"27c74670adb75075fad058d5ceaf7b20c4e7786c83bae8a32f626f9782af34c9" +
		"a33c2046ef60fd2a7878d378e29fec851806bbd9a67878f3a9f1cda4830763fd"
	OtherDigest512 digest.Digest = "sha512:" +
		"e25ac3845f8cbe12801a2dfa5a89d4c55dc47900f3b6edc9a9ee590f3c2b9312" +
		"f665d0039c93828b7b58f33950bc817a0955a9c5000a8d3e280569f08745ca68"
)

func seedBlob(t *testing.T, blobStore blobstore.BlobStore, d digest.Digest, content string) {
	t.Helper()
	_, err := blobStore.PutBlob(d, int64(len(content)), strings.NewReader(content))
	require.NoError(t, err)
}

func seedTestBlob(t *testing.T, blobStore blobstore.BlobStore) {
	t.Helper()
	seedBlob(t, blobStore, TestDigest, TestContent)
}
