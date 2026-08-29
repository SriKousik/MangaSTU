package format

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Registry holds all registered backup formats.
type Registry struct {
	formats []Format
}

// NewRegistry creates a new empty format registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a format to the registry.
func (r *Registry) Register(f Format) {
	r.formats = append(r.formats, f)
}

// Detect tries to identify the format of the file at path.
// It checks each registered format's Detect method.
func (r *Registry) Detect(path string) (Format, error) {
	for _, f := range r.formats {
		ok, err := f.Detect(path)
		if err != nil {
			continue
		}
		if ok {
			return f, nil
		}
	}
	return nil, fmt.Errorf("unrecognized backup format: %s", filepath.Base(path))
}

// GetByExtension finds a format that handles the given file extension.
func (r *Registry) GetByExtension(path string) (Format, error) {
	ext := strings.ToLower(filepath.Ext(path))
	for _, f := range r.formats {
		for _, fExt := range f.Extensions() {
			if strings.ToLower(fExt) == ext {
				return f, nil
			}
		}
	}
	return nil, fmt.Errorf("no format registered for extension: %s", ext)
}

// GetByName finds a format by its name (case-insensitive).
func (r *Registry) GetByName(name string) (Format, error) {
	lower := strings.ToLower(name)
	for _, f := range r.formats {
		if strings.ToLower(f.Name()) == lower {
			return f, nil
		}
	}
	return nil, fmt.Errorf("unknown format: %s", name)
}

// List returns all registered formats.
func (r *Registry) List() []Format {
	return r.formats
}
