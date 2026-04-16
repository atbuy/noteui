package sync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSameRevisionMatchesLegacyETagAgainstComposite(t *testing.T) {
	legacy := "etag-123"
	composite := buildWebDAVRevision(`"etag-123"`, []byte("original"))

	require.True(t, sameRevision(legacy, composite))
	require.True(t, sameRevision(composite, legacy))
}

func TestSameRevisionPrefersHashWhenBothAvailable(t *testing.T) {
	left := buildWebDAVRevision(`"etag-123"`, []byte("original"))
	right := buildWebDAVRevision(`"etag-123"`, []byte("edited remotely"))

	require.False(t, sameRevision(left, right))
}

func TestRemoteContentChangedUsesHashForLegacyETagRecords(t *testing.T) {
	rec := NoteRecord{
		RemoteRev:      "etag-123",
		LastSyncedHash: HashContent("original"),
	}
	remoteRevision := buildWebDAVRevision(`"etag-123"`, []byte("edited remotely"))

	require.True(t, remoteContentChanged(rec, remoteRevision))
}
