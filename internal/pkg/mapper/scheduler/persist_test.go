package scheduler_test

import (
	"context"
	"testing"

	"github.com/keboola/go-client/pkg/storageapi"
	"github.com/stretchr/testify/assert"

	"github.com/keboola/keboola-as-code/internal/pkg/model"
)

func TestVariablesMapBeforePersist(t *testing.T) {
	t.Parallel()
	state, d := createStateWithMapper(t)
	logger := d.DebugLogger()

	parentKey := model.ConfigKey{
		BranchID:    123,
		ComponentID: `foo.bar`,
		ID:          `345`,
	}
	configManifest := &model.ConfigManifest{
		ConfigKey: model.ConfigKey{
			BranchID:    123,
			ComponentID: storageapi.SchedulerComponentID,
			ID:          `678`,
		},
	}
	recipe := &model.PersistRecipe{
		ParentKey: parentKey,
		Manifest:  configManifest,
	}

	// Invoke
	assert.Empty(t, configManifest.Relations)
	assert.NoError(t, state.Mapper().MapBeforePersist(context.Background(), recipe))
	assert.Empty(t, logger.WarnAndErrorMessages())

	// Relation has been created
	assert.Equal(t, model.Relations{
		&model.SchedulerForRelation{
			ComponentID: `foo.bar`,
			ConfigID:    `345`,
		},
	}, configManifest.Relations)
}
