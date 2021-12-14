package links

import (
	"fmt"
	"sort"

	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
)

// onLocalLoad replaces shared code path by id in transformation config and blocks.
func (m *mapper) onLocalLoad(objectState model.ObjectState) error {
	// Shared code can be used only by transformation - struct must be set
	transformation, ok := objectState.LocalState().(*model.Config)
	if !ok || transformation.Transformation == nil {
		return nil
	}

	// Always remove shared code path from Content
	defer func() {
		transformation.Content.Delete(model.SharedCodePathContentKey)
	}()

	// Get shared code path
	sharedCodePathRaw, found := transformation.Content.Get(model.SharedCodePathContentKey)
	sharedCodePath, ok := sharedCodePathRaw.(string)
	if !found {
		return nil
	} else if !ok {
		return utils.PrefixError(
			fmt.Sprintf(`invalid transformation %s`, transformation.Desc()),
			fmt.Errorf(`key "%s" must be string, found %T`, model.SharedCodePathContentKey, sharedCodePathRaw),
		)
	}

	// Get shared code
	sharedCodeState, err := m.GetSharedCodeByPath(transformation.BranchKey(), sharedCodePath)
	if err != nil {
		return utils.PrefixError(
			err.Error(),
			fmt.Errorf(`referenced from %s`, objectState.Desc()),
		)
	} else if !sharedCodeState.HasLocalState() {
		return utils.PrefixError(
			fmt.Sprintf(`missing shared code %s`, sharedCodeState.Desc()),
			fmt.Errorf(`referenced from %s`, objectState.Desc()),
		)
	}
	sharedCodeConfig := sharedCodeState.LocalState().(*model.Config)
	if sharedCodeConfig.SharedCode == nil {
		// Value is not set, shared code is not valid -> skip
		return nil
	}

	// Check: target component of the shared code = transformation component
	if err := m.CheckTargetComponent(sharedCodeConfig, transformation.ConfigKey); err != nil {
		return err
	}

	// Store shared code config key in Transformation structure
	linkToSharedCode := &model.LinkToSharedCode{Config: sharedCodeState.ConfigKey}
	transformation.Transformation.LinkToSharedCode = linkToSharedCode

	// Replace paths -> IDs in code scripts
	errors := utils.NewMultiError()
	foundSharedCodeRows := make(map[model.RowId]model.ConfigRowKey)
	transformation.Transformation.MapScripts(func(code *model.Code, script model.Script) model.Script {
		if sharedCodeRow, v, err := m.replacePathByIdInScript(code, script, sharedCodeState); err != nil {
			errors.Append(err)
		} else if v != nil {
			foundSharedCodeRows[sharedCodeRow.Id] = sharedCodeRow.ConfigRowKey
			return v
		}
		return script
	})

	// Sort rows IDs
	for _, rowKey := range foundSharedCodeRows {
		linkToSharedCode.Rows = append(linkToSharedCode.Rows, rowKey)
	}
	sort.SliceStable(linkToSharedCode.Rows, func(i, j int) bool {
		return linkToSharedCode.Rows[i].String() < linkToSharedCode.Rows[j].String()
	})
	return errors.ErrorOrNil()
}
