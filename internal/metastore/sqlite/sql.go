package sqlite

const (
	sqlInsertNamespace = "INSERT OR IGNORE INTO namespace (name) VALUES (?)"
	sqlDeleteNamespace = "DELETE FROM namespace WHERE name = ?"
	sqlNamespaceExists = "SELECT EXISTS(SELECT 1 FROM namespace WHERE name = ?)"
)
