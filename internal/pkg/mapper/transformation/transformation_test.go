package transformation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem/aferofs"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/testapi"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
)

func createTestFixtures(t *testing.T, componentId string) (model.MapperContext, *model.Config, *model.ConfigManifest) {
	t.Helper()

	configKey := model.ConfigKey{
		BranchId:    123,
		ComponentId: componentId,
		Id:          `456`,
	}

	record := &model.ConfigManifest{
		ConfigKey: configKey,
		Paths: model.Paths{
			PathInProject: model.NewPathInProject(
				"branch",
				"config",
			),
		},
	}

	config := &model.Config{
		ConfigKey: configKey,
		Content:   utils.NewOrderedMap(),
	}

	logger, _ := utils.NewDebugLogger()
	fs, err := aferofs.NewMemoryFs(logger, ".")
	assert.NoError(t, err)

	state := model.NewState(zap.NewNop().Sugar(), fs, model.NewComponentsMap(testapi.NewMockedComponentsProvider()), model.SortByPath)
	context := model.MapperContext{Logger: logger, Fs: fs, Naming: model.DefaultNamingWithIds(), State: state}
	return context, config, record
}

func createLocalLoadRecipe(config *model.Config, configRecord *model.ConfigManifest) *model.LocalLoadRecipe {
	return &model.LocalLoadRecipe{
		Object:         config,
		ObjectManifest: configRecord,
		Metadata:       filesystem.NewJsonFile(model.MetaFile, utils.NewOrderedMap()),
		Configuration:  filesystem.NewJsonFile(model.ConfigFile, utils.NewOrderedMap()),
		Description:    filesystem.NewFile(model.DescriptionFile, ``),
	}
}

func createLocalSaveRecipe(config *model.Config, configRecord *model.ConfigManifest) *model.LocalSaveRecipe {
	return &model.LocalSaveRecipe{
		Object:         config,
		ObjectManifest: configRecord,
		Metadata:       filesystem.NewJsonFile(model.MetaFile, utils.NewOrderedMap()),
		Configuration:  filesystem.NewJsonFile(model.ConfigFile, utils.NewOrderedMap()),
		Description:    filesystem.NewFile(model.DescriptionFile, ``),
	}
}
