package metadata_test

import (
	"testing"

	"github.com/keboola/keboola-as-code/internal/pkg/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/mapper/template/metadata"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/state"
)

func createStateWithMapper(t *testing.T, templateRef model.TemplateRef, instanceId string, objectIds metadata.ObjectIdsMap, inputsUsage *metadata.InputsUsage) (*state.State, dependencies.Mocked) {
	t.Helper()
	d := dependencies.NewMockedDeps()
	mockedState := d.MockedState()
	mockedState.Mapper().AddMapper(metadata.NewMapper(mockedState, templateRef, instanceId, objectIds, inputsUsage))
	return mockedState, d
}
