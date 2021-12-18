package state

import (
	"context"
	"runtime"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/keboola/keboola-as-code/internal/pkg/env"
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/project/manifest"
	"github.com/keboola/keboola-as-code/internal/pkg/testapi"
	"github.com/keboola/keboola-as-code/internal/pkg/testhelper"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/orderedmap"
)

func TestLoadLocalStateMinimal(t *testing.T) {
	t.Parallel()

	m, fs := loadManifest(t, "minimal")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Empty(t, localErr)
	assert.Len(t, state.Branches(), 1)
	assert.Len(t, state.Configs(), 1)
	assert.Empty(t, state.UntrackedPaths())
	assert.Equal(t, []string{
		"main",
		"main/description.md",
		"main/extractor",
		"main/extractor/ex-generic-v2",
		"main/extractor/ex-generic-v2/456-todos",
		"main/extractor/ex-generic-v2/456-todos/config.json",
		"main/extractor/ex-generic-v2/456-todos/description.md",
		"main/extractor/ex-generic-v2/456-todos/meta.json",
		"main/meta.json",
	}, state.TrackedPaths())
}

func TestLoadLocalStateComplex(t *testing.T) {
	t.Parallel()

	m, fs := loadManifest(t, "complex")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Empty(t, localErr)
	assert.Equal(t, complexLocalExpectedBranches(), utils.SortByName(state.Branches()))
	assert.Equal(t, complexLocalExpectedConfigs(), utils.SortByName(state.Configs()))
	assert.Equal(t, complexLocalExpectedConfigRows(), utils.SortByName(state.ConfigRows()))
	assert.Equal(t, []string{
		"123-branch/extractor/ex-generic-v2/456-todos/untracked1",
		"123-branch/extractor/keboola.ex-db-mysql/untrackedDir",
		"123-branch/extractor/keboola.ex-db-mysql/untrackedDir/untracked2",
	}, state.UntrackedPaths())
	assert.Equal(t, []string{
		"123-branch",
		"123-branch/description.md",
		"123-branch/extractor",
		"123-branch/extractor/ex-generic-v2",
		"123-branch/extractor/ex-generic-v2/456-todos",
		"123-branch/extractor/ex-generic-v2/456-todos/config.json",
		"123-branch/extractor/ex-generic-v2/456-todos/description.md",
		"123-branch/extractor/ex-generic-v2/456-todos/meta.json",
		"123-branch/extractor/keboola.ex-db-mysql",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/config.json",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/description.md",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/meta.json",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/12-users",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/12-users/config.json",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/12-users/description.md",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/12-users/meta.json",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/34-test-view",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/34-test-view/config.json",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/34-test-view/description.md",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/34-test-view/meta.json",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/56-disabled",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/56-disabled/config.json",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/56-disabled/description.md",
		"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/56-disabled/meta.json",
		"123-branch/meta.json",
		"main",
		"main/description.md",
		"main/extractor",
		"main/extractor/ex-generic-v2",
		"main/extractor/ex-generic-v2/456-todos",
		"main/extractor/ex-generic-v2/456-todos/config.json",
		"main/extractor/ex-generic-v2/456-todos/description.md",
		"main/extractor/ex-generic-v2/456-todos/meta.json",
		"main/meta.json",
	}, state.TrackedPaths())
}

func TestLoadLocalStateAllowedBranches(t *testing.T) {
	t.Parallel()

	m, fs := loadManifest(t, "minimal")
	m.SetAllowedBranches(model.AllowedBranches{"main"})
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Empty(t, localErr)
}

func TestLoadLocalStateAllowedBranchesError(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "complex")
	m.SetAllowedBranches(model.AllowedBranches{"main"})
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Equal(t, `found manifest record for branch "123", but it is not allowed by the manifest definition`, localErr.Error())
}

func TestLoadLocalStateBranchMissingMetaJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "branch-missing-meta-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, `missing branch metadata file "main/meta.json"`, localErr.Error())
}

func TestLoadLocalStateBranchMissingDescription(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "branch-missing-description")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, `missing branch description file "main/description.md"`, localErr.Error())
}

func TestLoadLocalStateConfigMissingConfigJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-missing-config-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, `missing config file "123-branch/extractor/ex-generic-v2/456-todos/config.json"`, localErr.Error())
}

func TestLoadLocalStateConfigMissingMetaJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-missing-meta-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, `missing config metadata file "123-branch/extractor/ex-generic-v2/456-todos/meta.json"`, localErr.Error())
}

func TestLoadLocalStateConfigMissingDescription(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-missing-description")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, `missing config description file "123-branch/extractor/ex-generic-v2/456-todos/description.md"`, localErr.Error())
}

func TestLoadLocalStateConfigRowMissingConfigJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-row-missing-config-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, `missing config row file "123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/12-users/config.json"`, localErr.Error())
}

func TestLoadLocalStateConfigRowMissingMetaJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-row-missing-meta-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, `missing config row metadata file "123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/12-users/meta.json"`, localErr.Error())
}

func TestLoadLocalStateBranchInvalidMetaJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "branch-invalid-meta-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, "branch metadata file \"main/meta.json\" is invalid:\n  - invalid character 'f' looking for beginning of object key string, offset: 3", localErr.Error())
}

func TestLoadLocalStateConfigRowMissingDescription(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-row-missing-description")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, `missing config row description file "123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/12-users/description.md"`, localErr.Error())
}

func TestLoadLocalStateConfigInvalidConfigJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-invalid-config-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, "config file \"123-branch/extractor/ex-generic-v2/456-todos/config.json\" is invalid:\n  - invalid character 'f' looking for beginning of object key string, offset: 3", localErr.Error())
}

func TestLoadLocalStateConfigInvalidMetaJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-invalid-meta-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, "config metadata file \"123-branch/extractor/ex-generic-v2/456-todos/meta.json\" is invalid:\n  - invalid character 'f' looking for beginning of object key string, offset: 3", localErr.Error())
}

func TestLoadLocalStateConfigRowInvalidConfigJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-row-invalid-config-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, "config row file \"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/56-disabled/config.json\" is invalid:\n  - invalid character 'f' looking for beginning of object key string, offset: 3", localErr.Error())
}

func TestLoadLocalStateConfigRowInvalidMetaJson(t *testing.T) {
	t.Parallel()
	m, fs := loadManifest(t, "config-row-invalid-meta-json")
	state, localErr := loadLocalTestState(t, m, fs)
	assert.NotNil(t, state)
	assert.Error(t, localErr)
	assert.Equal(t, "config row metadata file \"123-branch/extractor/keboola.ex-db-mysql/896-tables/rows/12-users/meta.json\" is invalid:\n  - invalid character 'f' looking for beginning of object key string, offset: 3", localErr.Error())
}

func loadLocalTestState(t *testing.T, m *manifest.Manifest, fs filesystem.Fs) (*State, error) {
	t.Helper()

	// Mocked API
	logger := log.NewDebugLogger()
	api, httpTransport, _ := testapi.NewMockedStorageApi()

	// Mocked API response
	getGenericExResponder, err := httpmock.NewJsonResponder(200, map[string]interface{}{
		"id":                     "ex-generic-v2",
		"type":                   "extractor",
		"name":                   "Generic",
		"configurationSchema":    map[string]interface{}{},
		"configurationRowSchema": map[string]interface{}{},
	})
	assert.NoError(t, err)
	getMySqlExResponder, err := httpmock.NewJsonResponder(200, map[string]interface{}{
		"id":                     "keboola.ex-db-mysql",
		"type":                   "extractor",
		"name":                   "MySQL",
		"configurationSchema":    map[string]interface{}{},
		"configurationRowSchema": map[string]interface{}{},
	})
	assert.NoError(t, err)
	httpTransport.RegisterResponder("GET", `=~/storage/components/ex-generic-v2`, getGenericExResponder)
	httpTransport.RegisterResponder("GET", `=~/storage/components/keboola.ex-db-mysql`, getMySqlExResponder)

	// Load state
	schedulerApi, _, _ := testapi.NewMockedSchedulerApi()
	options := NewOptions(fs, m, api, schedulerApi, context.Background(), logger.Logger)
	options.LoadLocalState = true
	state, _, localErr, remoteErr := LoadState(options)
	assert.NoError(t, remoteErr)
	return state, localErr
}

func loadManifest(t *testing.T, projectDirName string) (*manifest.Manifest, filesystem.Fs) {
	t.Helper()

	envs := env.Empty()
	envs.Set("TEST_KBC_STORAGE_API_HOST", "foo.bar")
	envs.Set("LOCAL_PROJECT_ID", "12345")
	envs.Set("LOCAL_STATE_MAIN_BRANCH_ID", "111")
	envs.Set("LOCAL_STATE_MY_BRANCH_ID", "123")
	envs.Set("LOCAL_STATE_GENERIC_CONFIG_ID", "456")
	envs.Set("LOCAL_STATE_MYSQL_CONFIG_ID", "896")

	// State dir
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filesystem.Dir(testFile)
	stateDir := filesystem.Join(testDir, "..", "fixtures", "local", projectDirName)

	// Create Fs
	fs := testhelper.NewMemoryFsFrom(stateDir)
	testhelper.ReplaceEnvsDir(fs, `/`, envs)

	// Load manifest
	m, err := manifest.Load(fs)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	return m, fs
}

func complexLocalExpectedBranches() []*model.BranchState {
	return []*model.BranchState{
		{
			Local: &model.Branch{
				BranchKey: model.BranchKey{
					Id: 123,
				},
				Name:        "Branch",
				Description: "My branch",
				IsDefault:   false,
			},
			BranchManifest: &model.BranchManifest{
				RecordState: model.RecordState{
					Persisted: true,
				},
				BranchKey: model.BranchKey{
					Id: 123,
				},
				Paths: model.Paths{
					PathInProject: model.NewPathInProject(

						"",
						"123-branch",
					),
					RelatedPaths: []string{model.MetaFile, model.DescriptionFile},
				},
			},
		},
		{
			Local: &model.Branch{
				BranchKey: model.BranchKey{
					Id: 111,
				},
				Name:        "Main",
				Description: "Main branch",
				IsDefault:   true,
			},
			BranchManifest: &model.BranchManifest{
				RecordState: model.RecordState{
					Persisted: true,
				},
				BranchKey: model.BranchKey{
					Id: 111,
				},
				Paths: model.Paths{
					PathInProject: model.NewPathInProject(

						"",
						"main",
					),
					RelatedPaths: []string{model.MetaFile, model.DescriptionFile},
				},
			},
		},
	}
}

func complexLocalExpectedConfigs() []*model.ConfigState {
	return []*model.ConfigState{
		{
			Local: &model.Config{
				ConfigKey: model.ConfigKey{
					BranchId:    123,
					ComponentId: "keboola.ex-db-mysql",
					Id:          "896",
				},
				Name:              "tables",
				Description:       "tables config",
				ChangeDescription: "",
				Content: orderedmap.FromPairs([]orderedmap.Pair{
					{
						Key: "parameters",
						Value: orderedmap.FromPairs([]orderedmap.Pair{
							{
								Key: "db",
								Value: orderedmap.FromPairs([]orderedmap.Pair{
									{
										Key:   "host",
										Value: "mysql.example.com",
									},
								}),
							},
						}),
					},
				}),
			},
			ConfigManifest: &model.ConfigManifest{
				RecordState: model.RecordState{
					Persisted: true,
				},
				ConfigKey: model.ConfigKey{
					BranchId:    123,
					ComponentId: "keboola.ex-db-mysql",
					Id:          "896",
				},
				Paths: model.Paths{
					PathInProject: model.NewPathInProject(
						"123-branch",
						"extractor/keboola.ex-db-mysql/896-tables",
					),
					RelatedPaths: []string{model.MetaFile, model.ConfigFile, model.DescriptionFile},
				},
			},
		},
		{
			Local: &model.Config{
				ConfigKey: model.ConfigKey{
					BranchId:    111,
					ComponentId: "ex-generic-v2",
					Id:          "456",
				},
				Name:              "todos",
				Description:       "todos config",
				ChangeDescription: "",
				Content: orderedmap.FromPairs([]orderedmap.Pair{
					{
						Key: "parameters",
						Value: orderedmap.FromPairs([]orderedmap.Pair{
							{
								Key: "api",
								Value: orderedmap.FromPairs([]orderedmap.Pair{
									{
										Key:   "baseUrl",
										Value: "https://jsonplaceholder.typicode.com",
									},
								}),
							},
						}),
					},
				}),
			},
			ConfigManifest: &model.ConfigManifest{
				RecordState: model.RecordState{
					Persisted: true,
				},
				ConfigKey: model.ConfigKey{
					BranchId:    111,
					ComponentId: "ex-generic-v2",
					Id:          "456",
				},
				Paths: model.Paths{
					PathInProject: model.NewPathInProject(
						"main",
						"extractor/ex-generic-v2/456-todos",
					),
					RelatedPaths: []string{model.MetaFile, model.ConfigFile, model.DescriptionFile},
				},
			},
		},
		{
			Local: &model.Config{
				ConfigKey: model.ConfigKey{
					BranchId:    123,
					ComponentId: "ex-generic-v2",
					Id:          "456",
				},
				Name:              "todos",
				Description:       "todos config",
				ChangeDescription: "",
				Content: orderedmap.FromPairs([]orderedmap.Pair{
					{
						Key: "parameters",
						Value: orderedmap.FromPairs([]orderedmap.Pair{
							{
								Key: "api",
								Value: orderedmap.FromPairs([]orderedmap.Pair{
									{
										Key:   "baseUrl",
										Value: "https://jsonplaceholder.typicode.com",
									},
								}),
							},
						}),
					},
				}),
			},
			ConfigManifest: &model.ConfigManifest{
				RecordState: model.RecordState{
					Persisted: true,
				},
				ConfigKey: model.ConfigKey{
					BranchId:    123,
					ComponentId: "ex-generic-v2",
					Id:          "456",
				},
				Paths: model.Paths{
					PathInProject: model.NewPathInProject(
						"123-branch",
						"extractor/ex-generic-v2/456-todos",
					),
					RelatedPaths: []string{model.MetaFile, model.ConfigFile, model.DescriptionFile},
				},
			},
		},
	}
}

func complexLocalExpectedConfigRows() []*model.ConfigRowState {
	return []*model.ConfigRowState{
		{
			Local: &model.ConfigRow{
				ConfigRowKey: model.ConfigRowKey{
					BranchId:    123,
					ComponentId: "keboola.ex-db-mysql",
					ConfigId:    "896",
					Id:          "56",
				},
				Name:              "disabled",
				Description:       "",
				ChangeDescription: "",
				IsDisabled:        true,
				Content: orderedmap.FromPairs([]orderedmap.Pair{
					{
						Key: "parameters",
						Value: orderedmap.FromPairs([]orderedmap.Pair{
							{Key: "incremental", Value: false},
						}),
					},
				}),
			},
			ConfigRowManifest: &model.ConfigRowManifest{
				RecordState: model.RecordState{
					Persisted: true,
				},
				ConfigRowKey: model.ConfigRowKey{
					BranchId:    123,
					ComponentId: "keboola.ex-db-mysql",
					ConfigId:    "896",
					Id:          "56",
				},
				Paths: model.Paths{
					PathInProject: model.NewPathInProject(
						"123-branch/extractor/keboola.ex-db-mysql/896-tables",
						"rows/56-disabled",
					),
					RelatedPaths: []string{model.MetaFile, model.ConfigFile, model.DescriptionFile},
				},
			},
		},
		{
			Local: &model.ConfigRow{
				ConfigRowKey: model.ConfigRowKey{
					BranchId:    123,
					ComponentId: "keboola.ex-db-mysql",
					ConfigId:    "896",
					Id:          "34",
				},
				Name:              "test_view",
				Description:       "row description",
				ChangeDescription: "",
				IsDisabled:        false,
				Content: orderedmap.FromPairs([]orderedmap.Pair{
					{
						Key: "parameters",
						Value: orderedmap.FromPairs([]orderedmap.Pair{
							{Key: "incremental", Value: false},
						}),
					},
				}),
			},
			ConfigRowManifest: &model.ConfigRowManifest{
				RecordState: model.RecordState{
					Persisted: true,
				},
				ConfigRowKey: model.ConfigRowKey{
					BranchId:    123,
					ComponentId: "keboola.ex-db-mysql",
					ConfigId:    "896",
					Id:          "34",
				},
				Paths: model.Paths{
					PathInProject: model.NewPathInProject(
						"123-branch/extractor/keboola.ex-db-mysql/896-tables",
						"rows/34-test-view",
					),
					RelatedPaths: []string{model.MetaFile, model.ConfigFile, model.DescriptionFile},
				},
			},
		},
		{
			Local: &model.ConfigRow{
				ConfigRowKey: model.ConfigRowKey{
					BranchId:    123,
					ComponentId: "keboola.ex-db-mysql",
					ConfigId:    "896",
					Id:          "12",
				},
				Name:              "users",
				Description:       "",
				ChangeDescription: "",
				IsDisabled:        false,
				Content: orderedmap.FromPairs([]orderedmap.Pair{
					{
						Key: "parameters",
						Value: orderedmap.FromPairs([]orderedmap.Pair{
							{Key: "incremental", Value: false},
						}),
					},
				}),
			},
			ConfigRowManifest: &model.ConfigRowManifest{
				RecordState: model.RecordState{
					Persisted: true,
				},
				ConfigRowKey: model.ConfigRowKey{
					BranchId:    123,
					ComponentId: "keboola.ex-db-mysql",
					ConfigId:    "896",
					Id:          "12",
				},
				Paths: model.Paths{
					PathInProject: model.NewPathInProject(
						"123-branch/extractor/keboola.ex-db-mysql/896-tables",
						"rows/12-users",
					),
					RelatedPaths: []string{model.MetaFile, model.ConfigFile, model.DescriptionFile},
				},
			},
		},
	}
}
