package ports

// FileWriter defines the interface for file system write operations
type FileWriter interface {
	// Write writes content to a file at the specified path
	// Creates parent directories if they don't exist
	Write(path string, content []byte) error

	// Exists checks if a file exists at the specified path
	Exists(path string) bool
}
