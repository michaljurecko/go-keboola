package defaultbucket

import (
	"fmt"

	"github.com/keboola/keboola-as-code/internal/pkg/local"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/orderedmap"
)

type defaultBucketMapper struct {
	model.MapperContext
	localManager *local.Manager
}

type configOrRow interface {
	model.ObjectWithContent
	BranchKey() model.BranchKey
}

func NewMapper(localManager *local.Manager, context model.MapperContext) *defaultBucketMapper {
	return &defaultBucketMapper{
		MapperContext: context,
		localManager:  localManager,
	}
}

func (m *defaultBucketMapper) visitStorageInputTables(config configOrRow, content *orderedmap.OrderedMap, callback func(
	config configOrRow,
	sourceTableId string,
	storageInputTable *orderedmap.OrderedMap,
) error) error {
	inputTablesRaw, _, _ := content.GetNested("storage.input.tables")
	inputTables, ok := inputTablesRaw.([]interface{})
	if !ok {
		return nil
	}

	for _, inputTableRaw := range inputTables {
		inputTable, ok := inputTableRaw.(*orderedmap.OrderedMap)
		if !ok {
			continue
		}
		inputTableSourceRaw, ok := inputTable.Get(`source`)
		if !ok {
			continue
		}
		inputTableSource, ok := inputTableSourceRaw.(string)
		if !ok {
			continue
		}

		err := callback(config, inputTableSource, inputTable)
		if err != nil {
			return err
		}
	}

	return nil
}

func markUsedInInputMapping(omConfig *model.Config, usedIn configOrRow) {
	switch v := usedIn.(type) {
	case *model.Config:
		omConfig.Relations.Add(&model.UsedInConfigInputMappingRelation{
			UsedIn: v.ConfigKey,
		})
	case *model.ConfigRow:
		omConfig.Relations.Add(&model.UsedInRowInputMappingRelation{
			UsedIn: v.ConfigRowKey,
		})
	default:
		panic(fmt.Errorf(`unexpected type "%T"`, usedIn))
	}
}
