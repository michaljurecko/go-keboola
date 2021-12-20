package template

import (
	"github.com/keboola/keboola-as-code/internal/pkg/model"
)

type template struct {
	objects []model.Object
}

func FromState(objects model.ObjectStates, stateType model.StateType) *template {
	return &template{objects: objectFromState(objects, stateType)}
}

func (t *template) ReplaceKeys(keys KeysReplacement) error {
	values, err := keys.Values()
	if err != nil {
		return err
	}
	t.objects = replaceValues(values, t.objects).([]model.Object)
	return nil
}

func objectFromState(allObjects model.ObjectStates, stateType model.StateType) []model.Object {
	all := allObjects.ObjectsInState(stateType).All()
	objects := make([]model.Object, len(all))
	copy(objects, all)
	return objects
}
