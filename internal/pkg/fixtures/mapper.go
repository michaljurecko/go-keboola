package fixtures

import (
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/orderedmap"
)

func NewLocalLoadRecipe(manifest model.ObjectManifest, object model.Object) *model.LocalLoadRecipe {
	recipe := &model.LocalLoadRecipe{
		Object:         object,
		ObjectManifest: manifest,
	}

	recipe.Files.
		Add(filesystem.NewJsonFile(`meta.json`, orderedmap.New())).
		AddTag(model.FileTypeJson).
		AddTag(model.FileKindObjectMeta)

	recipe.Files.
		Add(filesystem.NewJsonFile(`config.json`, orderedmap.New())).
		AddTag(model.FileTypeJson).
		AddTag(model.FileKindObjectConfig)

	recipe.Files.
		Add(filesystem.NewFile(`description.md`, ``)).
		AddTag(model.FileTypeMarkdown).
		AddTag(model.FileKindObjectDescription)

	return recipe
}

func NewLocalSaveRecipe(manifest model.ObjectManifest, object model.Object) *model.LocalSaveRecipe {
	recipe := &model.LocalSaveRecipe{
		Object:         object,
		ObjectManifest: manifest,
	}

	recipe.Files.
		Add(filesystem.NewJsonFile(`meta.json`, orderedmap.New())).
		AddTag(model.FileTypeJson).
		AddTag(model.FileKindObjectMeta)

	recipe.Files.
		Add(filesystem.NewJsonFile(`config.json`, orderedmap.New())).
		AddTag(model.FileTypeJson).
		AddTag(model.FileKindObjectConfig)

	recipe.Files.
		Add(filesystem.NewFile(`description.md`, ``)).
		AddTag(model.FileTypeMarkdown).
		AddTag(model.FileKindObjectDescription)

	return recipe
}
