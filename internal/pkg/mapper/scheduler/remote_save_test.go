package scheduler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/keboola/keboola-as-code/internal/pkg/json"
	. "github.com/keboola/keboola-as-code/internal/pkg/mapper/scheduler"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/testapi"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/orderedmap"
)

func TestSchedulerMapBeforeRemoteSave(t *testing.T) {
	t.Parallel()
	context := createMapperContext(t)
	schedulerApi, _, _ := testapi.NewMockedSchedulerApi()
	mapper := NewMapper(context, schedulerApi)

	// Scheduler config
	content := orderedmap.New()
	json.MustDecodeString(`{"target": {"mode": "run"}}`, content)
	apiObject := &model.Config{Content: content}
	apiObject.AddRelation(&model.SchedulerForRelation{
		ComponentId: `foo.bar`,
		ConfigId:    `12345`,
	})
	internalObject := apiObject.Clone().(*model.Config)
	recipe := &model.RemoteSaveRecipe{
		ApiObject:      apiObject,
		InternalObject: internalObject,
		ObjectManifest: &model.ConfigManifest{},
	}

	// Invoke
	assert.NotEmpty(t, apiObject.Relations)
	assert.NotEmpty(t, internalObject.Relations)
	assert.NoError(t, mapper.MapBeforeRemoteSave(recipe))

	// Internal object is not changed
	assert.Equal(t, model.Relations{
		&model.SchedulerForRelation{
			ComponentId: `foo.bar`,
			ConfigId:    `12345`,
		},
	}, internalObject.Relations)
	targetInternalRaw, found := internalObject.Content.Get(model.SchedulerTargetKey)
	assert.True(t, found)
	targetInternal, ok := targetInternalRaw.(*orderedmap.OrderedMap)
	assert.True(t, ok)
	_, found = targetInternal.Get(model.SchedulerTargetComponentIdKey)
	assert.False(t, found)
	_, found = targetInternal.Get(model.SchedulerTargetConfigurationIdKey)
	assert.False(t, found)

	// All relations have been mapped
	assert.Empty(t, apiObject.Relations)

	// Api object contains target
	targetRaw, found := apiObject.Content.Get(model.SchedulerTargetKey)
	assert.True(t, found)
	target, ok := targetRaw.(*orderedmap.OrderedMap)
	assert.True(t, ok)

	// Api object contains componentId and configurationId
	componentId, found := target.Get(model.SchedulerTargetComponentIdKey)
	assert.True(t, found)
	assert.Equal(t, `foo.bar`, componentId)
	configurationId, found := target.Get(model.SchedulerTargetConfigurationIdKey)
	assert.True(t, found)
	assert.Equal(t, `12345`, configurationId)
}
