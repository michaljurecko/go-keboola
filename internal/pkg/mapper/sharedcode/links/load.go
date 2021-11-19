package links

import (
	"fmt"
	"sort"

	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
)

func (m *mapper) OnObjectsLoad(event model.OnObjectsLoadEvent) error {
	// Only on local load
	if event.StateType != model.StateTypeLocal {
		return nil
	}

	// Check all new objects
	errors := utils.NewMultiError()
	for _, object := range event.NewObjects {
		if err := m.replaceSharedCodePathById(object); err != nil {
			errors.Append(err)
		}
	}
	return errors.ErrorOrNil()
}

// replaceSharedCodePathById in transformation config + blocks.
func (m *mapper) replaceSharedCodePathById(object model.Object) error {
	transformation, sharedCodePath, err := m.GetSharedCodePath(object)
	if err != nil {
		return err
	} else if transformation == nil {
		return nil
	}

	// Remove shared code id
	defer func() {
		transformation.Content.Delete(model.SharedCodePathContentKey)
	}()

	// Get shared code transformation
	sharedCodeState, err := m.GetSharedCodeByPath(transformation.BranchKey(), sharedCodePath)
	if err != nil {
		errors := utils.NewMultiError()
		errors.Append(err)
		errors.AppendRaw(fmt.Sprintf(`  - referenced from %s`, transformation.Desc()))
		return errors
	}
	sharedCode := sharedCodeState.LocalOrRemoteState().(*model.Config)
	targetComponentId, err := m.GetTargetComponentId(sharedCode)
	if err != nil {
		return err
	}

	// Check componentId
	if targetComponentId != transformation.ComponentId {
		errors := utils.NewMultiError()
		errors.Append(fmt.Errorf(`unexpected shared code "%s" in %s`, model.SharedCodeComponentIdContentKey, sharedCodeState.Desc()))
		errors.AppendRaw(fmt.Sprintf(`  - expected "%s"`, transformation.ComponentId))
		errors.AppendRaw(fmt.Sprintf(`  - found "%s"`, targetComponentId))
		errors.AppendRaw(fmt.Sprintf(`  - referenced from %s`, transformation.Desc()))
		return errors
	}

	// Replace Shared Code Path -> Shared Code ID
	transformation.Content.Set(model.SharedCodeIdContentKey, sharedCodeState.Id)

	// Replace IDs -> paths in scripts
	errors := utils.NewMultiError()
	rowIdsMap := make(map[string]bool)
	for _, block := range transformation.Blocks {
		for _, code := range block.Codes {
			for index, script := range code.Scripts {
				if id, v, err := m.replacePathByIdInScript(script, code, sharedCodeState); err != nil {
					errors.Append(err)
					continue
				} else if v != "" {
					rowIdsMap[id] = true
					code.Scripts[index] = v
				}
			}
		}
	}

	// Convert row IDs map -> slice
	rowIds := make([]interface{}, 0)
	for id := range rowIdsMap {
		rowIds = append(rowIds, id)
	}
	sort.SliceStable(rowIds, func(i, j int) bool {
		return rowIds[i].(string) < rowIds[j].(string)
	})

	// Set rows IDs
	transformation.Content.Set(model.SharedCodeRowsIdContentKey, rowIds)
	return errors.ErrorOrNil()
}

// replacePathByIdInScript from transformation code.
func (m *mapper) replacePathByIdInScript(script string, code *model.Code, sharedCode *model.ConfigState) (string, string, error) {
	path := m.matchPath(script, code.ComponentId)
	if path == "" {
		// Not found
		return "", "", nil
	}

	// Get shared code config row
	row, err := m.GetSharedCodeRowByPath(sharedCode, path)
	if err != nil {
		errors := utils.NewMultiError()
		errors.Append(err)
		errors.AppendRaw(fmt.Sprintf(`  - referenced from "%s"`, code.Path()))
		return "", "", errors
	}

	// Return ID instead of path
	return row.Id, m.formatId(row.Id), nil
}
