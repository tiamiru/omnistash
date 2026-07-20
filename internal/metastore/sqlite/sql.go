package sqlite

const (
	sqlInsertNamespace = "INSERT OR IGNORE INTO namespace (name) VALUES (?)"
	sqlDeleteNamespace = "DELETE FROM namespace WHERE name = ?"
	sqlSelectNamespace = "SELECT name, created_at, updated_at FROM namespace WHERE name = ?"
	sqlNamespaceExists = "SELECT EXISTS(SELECT 1 FROM namespace WHERE name = ?)"
)

const (
	sqlGetNamespaceBlobSize = `
		SELECT size FROM namespace_blobs
		WHERE name = ? AND digest = ? AND lifecycle = 'active'`

	sqlResurrectNamespaceBlob = `
		UPDATE namespace_blobs SET lifecycle='active', deleted_at=NULL
		WHERE name = ? AND digest = ? AND lifecycle = 'pending_deletion'`

	sqlInsertNamespaceBlob = `
		INSERT OR IGNORE INTO namespace_blobs (name, digest, size)
		VALUES (?, ?, ?)`
)

const (
	sqlInsertManifest = `
		INSERT OR IGNORE INTO manifests (namespace, digest, media_type, size)
		VALUES (?, ?, ?, ?)`

	sqlGetManifestByDigest = `
		SELECT namespace, digest, media_type, size FROM manifests
		WHERE namespace = ? AND digest = ? AND lifecycle = 'active'`

	sqlDeleteManifestByDigest = `
		UPDATE manifests SET lifecycle = 'pending_deletion', deleted_at = unixepoch()
		WHERE namespace = ? AND digest = ? AND lifecycle = 'active'`
)
