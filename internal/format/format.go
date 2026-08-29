package format

import (
	"time"

	"github.com/misfit/mangastu/internal/core"
)

// ReadResult contains the parsed backup along with metadata.
type ReadResult struct {
	Backup    *core.Backup
	CreatedAt time.Time // when the backup was originally created
	Warnings  []core.Warning
}

// Format defines the interface that every backup format must implement.
type Format interface {
	// Name returns the human-readable format name.
	Name() string

	// Extensions returns the file extensions this format uses (e.g., ".tmb", ".tachibk").
	Extensions() []string

	// Detect inspects a file and returns true if it matches this format.
	Detect(path string) (bool, error)

	// Read parses the backup file at path and returns the internal model.
	Read(path string) (*ReadResult, error)

	// Write serializes the internal model to the format and writes to path.
	Write(path string, backup *core.Backup) error
}
