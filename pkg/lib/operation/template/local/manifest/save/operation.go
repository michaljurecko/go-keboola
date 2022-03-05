package save

import (
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/template"
)

type Dependencies interface {
	Logger() log.Logger
}

func Run(m *template.Manifest, fs filesystem.Fs, d Dependencies) (changed bool, err error) {
	// Save if manifest is changed
	if m.IsChanged() {
		if err := m.Save(fs); err != nil {
			return false, err
		}
		return true, nil
	}

	d.Logger().Debugf(`Template manifest has not changed.`)
	return false, nil
}
