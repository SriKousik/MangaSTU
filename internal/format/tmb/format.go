package tmb

import (
	"github.com/misfit/mangastu/internal/format"
)

// TMB implements the Format interface for Tachimanga backup files (.tmb).
type TMB struct{}

var _ format.Format = (*TMB)(nil)

func (t *TMB) Name() string {
	return "Tachimanga Backup"
}

func (t *TMB) Extensions() []string {
	return []string{".tmb"}
}
