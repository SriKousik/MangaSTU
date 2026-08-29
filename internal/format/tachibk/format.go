package tachibk

import (
	"github.com/misfit/mangastu/internal/format"
)

// TachiBK implements the Format interface for Komikku/Mihon backup files (.tachibk).
type TachiBK struct{}

var _ format.Format = (*TachiBK)(nil)

func (t *TachiBK) Name() string {
	return "Tachiyomi/Komikku Backup"
}

func (t *TachiBK) Extensions() []string {
	return []string{".tachibk"}
}
