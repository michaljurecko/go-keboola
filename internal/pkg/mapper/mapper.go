package mapper

import (
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/naming"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
)

type Context struct {
	Logger          log.Logger
	Fs              filesystem.Fs
	NamingGenerator *naming.Generator
	NamingRegistry  *naming.Registry
	State           model.ObjectStates
}

// LocalSaveMapper to modify how the object will be saved in the filesystem.
type LocalSaveMapper interface {
	MapBeforeLocalSave(recipe *model.LocalSaveRecipe) error
}

// LocalLoadMapper to modify/normalize the object internal representation after loading from the filesystem.
// Note: do not rely on other objects, they may not be loaded yet, see OnLocalChangeListener.
type LocalLoadMapper interface {
	MapAfterLocalLoad(recipe *model.LocalLoadRecipe) error
}

// RemoteSaveMapper to modify how the object will be saved in the Storage API.
type RemoteSaveMapper interface {
	MapBeforeRemoteSave(recipe *model.RemoteSaveRecipe) error
}

// RemoteLoadMapper to modify/normalize the object internal representation after loading from the Storage API.
// Note: do not rely on other objects, they may not be loaded yet, see OnRemoteChangeListener.
type RemoteLoadMapper interface {
	MapAfterRemoteLoad(recipe *model.RemoteLoadRecipe) error
}

// BeforePersistMapper to modify manifest record before persist.
type BeforePersistMapper interface {
	MapBeforePersist(recipe *model.PersistRecipe) error
}

// OnObjectPathUpdateListener is called when a local path has been updated.
type OnObjectPathUpdateListener interface {
	OnObjectPathUpdate(event model.OnObjectPathUpdateEvent) error
}

type OnLocalChangeListener interface {
	OnLocalChange(changes *model.LocalChanges) error
}

type OnRemoteChangeListener interface {
	OnRemoteChange(changes *model.RemoteChanges) error
}

type Mappers []interface{}

func (m Mappers) ForEach(stopOnFailure bool, callback func(mapper interface{}) error) error {
	errors := utils.NewMultiError()
	for _, mapper := range m {
		if err := callback(mapper); err != nil {
			if stopOnFailure {
				return err
			}
			errors.Append(err)
		}
	}
	return errors.ErrorOrNil()
}

func (m Mappers) ForEachReverse(stopOnFailure bool, callback func(mapper interface{}) error) error {
	errors := utils.NewMultiError()
	l := len(m)
	for i := l - 1; i >= 0; i-- {
		if err := callback(m[i]); err != nil {
			if stopOnFailure {
				return err
			}
			errors.Append(err)
		}
	}
	return errors.ErrorOrNil()
}

// Mapper maps Objects between internal/filesystem/API representations.
//
// The mappers are called in the order in which they were entered (Mappers.ForEach).
// Except for save methods: MapBeforeLocalSave, MapBeforeRemoteSave.
// For these, the mappers are called in reverse order (Mappers.ForEachReverse).
type Mapper struct {
	mappers Mappers // implement part of the interfaces above
}

func New() *Mapper {
	return &Mapper{}
}

func (m *Mapper) AddMapper(mapper ...interface{}) *Mapper {
	m.mappers = append(m.mappers, mapper...)
	return m
}

func (m *Mapper) MapBeforeLocalSave(recipe *model.LocalSaveRecipe) error {
	return m.mappers.ForEachReverse(true, func(mapper interface{}) error {
		if mapper, ok := mapper.(LocalSaveMapper); ok {
			if err := mapper.MapBeforeLocalSave(recipe); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Mapper) MapAfterLocalLoad(recipe *model.LocalLoadRecipe) error {
	return m.mappers.ForEach(true, func(mapper interface{}) error {
		if mapper, ok := mapper.(LocalLoadMapper); ok {
			if err := mapper.MapAfterLocalLoad(recipe); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Mapper) MapBeforeRemoteSave(recipe *model.RemoteSaveRecipe) error {
	return m.mappers.ForEachReverse(true, func(mapper interface{}) error {
		if mapper, ok := mapper.(RemoteSaveMapper); ok {
			if err := mapper.MapBeforeRemoteSave(recipe); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Mapper) MapAfterRemoteLoad(recipe *model.RemoteLoadRecipe) error {
	return m.mappers.ForEach(true, func(mapper interface{}) error {
		if mapper, ok := mapper.(RemoteLoadMapper); ok {
			if err := mapper.MapAfterRemoteLoad(recipe); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Mapper) MapBeforePersist(recipe *model.PersistRecipe) error {
	return m.mappers.ForEach(false, func(mapper interface{}) error {
		if mapper, ok := mapper.(BeforePersistMapper); ok {
			if err := mapper.MapBeforePersist(recipe); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Mapper) OnObjectPathUpdate(event model.OnObjectPathUpdateEvent) error {
	return m.mappers.ForEach(false, func(mapper interface{}) error {
		if mapper, ok := mapper.(OnObjectPathUpdateListener); ok {
			if err := mapper.OnObjectPathUpdate(event); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Mapper) OnLocalChange(changes *model.LocalChanges) error {
	return m.mappers.ForEach(false, func(mapper interface{}) error {
		if mapper, ok := mapper.(OnLocalChangeListener); ok {
			if err := mapper.OnLocalChange(changes); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Mapper) OnRemoteChange(changes *model.RemoteChanges) error {
	return m.mappers.ForEach(false, func(mapper interface{}) error {
		if mapper, ok := mapper.(OnRemoteChangeListener); ok {
			if err := mapper.OnRemoteChange(changes); err != nil {
				return err
			}
		}
		return nil
	})
}
