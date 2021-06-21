package model

import (
	"github.com/iancoleman/strcase"
	"github.com/spf13/cast"
	"path/filepath"
	"regexp"
	"strings"
)

type LocalNaming struct {
	Branch    string `json:"branch" validate:"required"`
	Config    string `json:"config" validate:"required"`
	ConfigRow string `json:"configRow" validate:"required"`
}

func DefaultNaming() *LocalNaming {
	return &LocalNaming{
		Branch:    "{branch_id}-{branch_name}",
		Config:    "{component_type}/{component_id}/{config_id}-{config_name}",
		ConfigRow: "{config_row_id}-{config_row_name}",
	}
}

func (n *LocalNaming) BranchPath(branch *Branch) string {
	return n.replace(n.Branch, map[string]interface{}{
		"branch_id":   branch.Id,
		"branch_name": n.normalizeName(branch.Name),
	})
}

func (n *LocalNaming) ConfigPath(component *Component, config *Config) string {
	return n.replace(n.Config, map[string]interface{}{
		"component_type": component.Type,
		"component_id":   component.Id,
		"config_id":      config.Id,
		"config_name":    n.normalizeName(config.Name),
	})
}

func (n *LocalNaming) ConfigRowPath(row *ConfigRow) string {
	return n.replace(n.ConfigRow, map[string]interface{}{
		"config_row_id":   row.Id,
		"config_row_name": n.normalizeName(row.Name),
	})
}

func (n *LocalNaming) normalizeName(name string) string {
	return regexp.
		MustCompile(`[^a-zA-Z0-9-]]`).
		ReplaceAllString(strcase.ToDelimited(name, '-'), "-")
}

func (n *LocalNaming) replace(path string, placeholders map[string]interface{}) string {
	path = strings.ReplaceAll(path, "/", string(filepath.Separator))
	for key, value := range placeholders {
		path = strings.ReplaceAll(path, "{"+key+"}", cast.ToString(value))
	}
	return path
}
