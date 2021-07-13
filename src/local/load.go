package local

import (
	"fmt"
	"keboola-as-code/src/json"
	"keboola-as-code/src/manifest"
	"keboola-as-code/src/model"
	"keboola-as-code/src/utils"
	"path/filepath"
	"reflect"
)

// LoadModel from manifest and disk
func LoadModel(projectDir string, record manifest.Record, target interface{}) (found bool, err error) {
	errors := &utils.Error{}

	// Check if directory exists
	if !utils.IsDir(filepath.Join(projectDir, record.RelativePath())) {
		errors.Add(fmt.Errorf(`%s "%s" not found`, record.Kind().Name, record.RelativePath()))
		return false, errors
	}

	// Load values from the meta file
	errPrefix := record.Kind().Name + " metadata"
	if err := utils.ReadTaggedFields(projectDir, record.MetaFilePath(), model.MetaFileTag, errPrefix, target); err != nil {
		errors.Add(err)
	}

	// Load config file content
	errPrefix = record.Kind().Name
	if configField := utils.GetOneFieldWithTag(model.ConfigFileTag, target); configField != nil {
		content := utils.NewOrderedMap()
		modelValue := reflect.ValueOf(target).Elem()
		if err := json.ReadFile(projectDir, record.ConfigFilePath(), &content, errPrefix); err == nil {
			modelValue.FieldByName(configField.Name).Set(reflect.ValueOf(content))
		} else {
			errors.Add(err)
		}
	}

	if errors.Len() > 0 {
		return true, errors
	}

	return true, nil
}

func LoadBranch(projectDir string, b *manifest.BranchManifest) (branch *model.Branch, found bool, err error) {
	branch = &model.Branch{BranchKey: b.BranchKey}
	found, err = LoadModel(projectDir, b, branch)
	if err != nil {
		return nil, found, err
	}
	return
}

func LoadConfig(projectDir string, c *manifest.ConfigManifest) (config *model.Config, found bool, err error) {
	config = &model.Config{ConfigKey: c.ConfigKey}
	found, err = LoadModel(projectDir, c, config)
	if err != nil {
		return nil, found, err
	}
	return
}

func LoadConfigRow(projectDir string, r *manifest.ConfigRowManifest) (row *model.ConfigRow, found bool, err error) {
	row = &model.ConfigRow{ConfigRowKey: r.ConfigRowKey}
	found, err = LoadModel(projectDir, r, row)
	if err != nil {
		return nil, found, err
	}
	return
}
