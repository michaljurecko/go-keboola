package state

import (
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"keboola-as-code/src/model"
	"keboola-as-code/src/remote"
	"keboola-as-code/src/utils"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadLocalStateMinimal(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "minimal")
	assert.NotNil(t, state)
	assert.Equal(t, 0, state.LocalErrors().Len())
	assert.Len(t, state.Branches(), 1)
	assert.Len(t, state.Configs(), 1)
	assert.Empty(t, state.UntrackedPaths())
	assert.Equal(t, []string{
		"main",
		"main/ex-generic-v2",
		"main/ex-generic-v2/456-todos",
		"main/ex-generic-v2/456-todos/config.json",
		"main/ex-generic-v2/456-todos/meta.json",
		"main/meta.json",
	}, state.TrackedPaths())
}

func TestLoadLocalStateComplex(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "complex")
	assert.NotNil(t, state)
	assert.Equal(t, 0, state.LocalErrors().Len())
	assert.Equal(t, complexLocalExpectedBranches(), state.Branches())
	assert.Equal(t, complexLocalExpectedConfigs(), state.Configs())
	assert.Equal(t, complexLocalExpectedConfigRows(), state.ConfigRows())
	assert.Equal(t, []string{
		"123-branch/keboola.ex-db-mysql/untrackedDir",
		"123-branch/keboola.ex-db-mysql/untrackedDir/untracked2",
		"123-branch/keboola.ex-generic/456-todos/untracked1",
	}, state.UntrackedPaths())
	assert.Equal(t, []string{
		"123-branch",
		"123-branch/keboola.ex-db-mysql",
		"123-branch/keboola.ex-db-mysql/896-tables",
		"123-branch/keboola.ex-db-mysql/896-tables/config.json",
		"123-branch/keboola.ex-db-mysql/896-tables/meta.json",
		"123-branch/keboola.ex-db-mysql/896-tables/rows",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/12-users",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/12-users/config.json",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/12-users/meta.json",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/34-test-view",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/34-test-view/config.json",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/34-test-view/meta.json",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/56-disabled",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/56-disabled/config.json",
		"123-branch/keboola.ex-db-mysql/896-tables/rows/56-disabled/meta.json",
		"123-branch/keboola.ex-generic",
		"123-branch/keboola.ex-generic/456-todos",
		"123-branch/keboola.ex-generic/456-todos/config.json",
		"123-branch/keboola.ex-generic/456-todos/meta.json",
		"123-branch/meta.json",
		"main",
		"main/keboola.ex-generic",
		"main/keboola.ex-generic/456-todos",
		"main/keboola.ex-generic/456-todos/config.json",
		"main/keboola.ex-generic/456-todos/meta.json",
		"main/meta.json",
	}, state.TrackedPaths())
}

func TestLoadLocalStateBranchMissingMetaJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "branch-missing-meta-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `branch metadata file "main/meta.json" not found`, state.LocalErrors().Error())
}

func TestLoadLocalStateConfigMissingConfigJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "config-missing-config-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `config file "123-branch/keboola.ex-generic/456-todos/config.json" not found`, state.LocalErrors().Error())
}

func TestLoadLocalStateConfigMissingMetaJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "config-missing-meta-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `config metadata file "123-branch/keboola.ex-generic/456-todos/meta.json" not found`, state.LocalErrors().Error())
}

func TestLoadLocalStateConfigRowMissingConfigJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "config-row-missing-config-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `config row file "123-branch/keboola.ex-db-mysql/896-tables/rows/12-users/config.json" not found`, state.LocalErrors().Error())
}

func TestLoadLocalStateConfigRowMissingMetaJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "config-row-missing-meta-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `config row metadata file "123-branch/keboola.ex-db-mysql/896-tables/rows/12-users/meta.json" not found`, state.LocalErrors().Error())
}

func TestLoadLocalStateBranchInvalidMetaJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "branch-invalid-meta-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `branch metadata file "main/meta.json" is invalid: invalid character 'f' looking for beginning of object key string, offset: 3`, state.LocalErrors().Error())
}

func TestLoadLocalStateConfigInvalidConfigJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "config-invalid-config-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `config file "123-branch/keboola.ex-generic/456-todos/config.json" is invalid: invalid character 'f' looking for beginning of object key string, offset: 3`, state.LocalErrors().Error())
}

func TestLoadLocalStateConfigInvalidMetaJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "config-invalid-meta-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `config metadata file "123-branch/keboola.ex-generic/456-todos/meta.json" is invalid: invalid character 'f' looking for beginning of object key string, offset: 3`, state.LocalErrors().Error())
}

func TestLoadLocalStateConfigRowInvalidConfigJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "config-row-invalid-config-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `config row file "123-branch/keboola.ex-db-mysql/896-tables/rows/56-disabled/config.json" is invalid: invalid character 'f' looking for beginning of object key string, offset: 3`, state.LocalErrors().Error())
}

func TestLoadLocalStateConfigRowInvalidMetaJson(t *testing.T) {
	defer utils.ResetEnv(t, os.Environ())
	state := loadLocalTestState(t, "config-row-invalid-meta-json")
	assert.NotNil(t, state)
	assert.Greater(t, state.LocalErrors().Len(), 0)
	assert.Equal(t, `config row metadata file "123-branch/keboola.ex-db-mysql/896-tables/rows/12-users/meta.json" is invalid: invalid character 'f' looking for beginning of object key string, offset: 3`, state.LocalErrors().Error())
}

func loadLocalTestState(t *testing.T, projectDirName string) *model.State {
	utils.MustSetEnv("LOCAL_STATE_MAIN_BRANCH_ID", "111")
	utils.MustSetEnv("LOCAL_STATE_MY_BRANCH_ID", "123")
	utils.MustSetEnv("LOCAL_STATE_GENERIC_CONFIG_ID", "456")
	utils.MustSetEnv("LOCAL_STATE_MYSQL_CONFIG_ID", "896")

	api, _ := remote.TestStorageApiWithToken(t)

	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	stateDir := filepath.Join(testDir, "..", "fixtures", "local", projectDirName)
	projectDir := t.TempDir()
	metadataDir := filepath.Join(projectDir, model.MetadataDir)

	err := copy.Copy(stateDir, projectDir)
	if err != nil {
		t.Fatalf("Copy error: %s", err)
	}
	utils.ReplaceEnvsDir(projectDir)

	state := model.NewState(projectDir, model.DefaultNaming())
	manifest, err := model.LoadManifest(projectDir, metadataDir)
	if err != nil {
		assert.FailNow(t, state.LocalErrors().Error())
	}

	LoadLocalState(state, manifest, api)
	return state
}

func complexLocalExpectedBranches() []*model.BranchState {
	return []*model.BranchState{
		{
			Local: &model.Branch{
				Id:          111,
				Name:        "Main",
				Description: "Main branch",
				IsDefault:   true,
			},
			BranchManifest: &model.BranchManifest{
				ManifestPaths: model.ManifestPaths{
					Path:       "main",
					ParentPath: "",
				},
				Id: 111,
			},
		},
		{
			Local: &model.Branch{
				Id:          123,
				Name:        "Branch",
				Description: "My branch",
				IsDefault:   false,
			},
			BranchManifest: &model.BranchManifest{
				ManifestPaths: model.ManifestPaths{
					Path:       "123-branch",
					ParentPath: "",
				},
				Id: 123,
			},
		},
	}
}

func complexLocalExpectedConfigs() []*model.ConfigState {
	return []*model.ConfigState{
		{
			Local: &model.Config{
				BranchId:          111,
				ComponentId:       "keboola.ex-generic",
				Id:                "456",
				Name:              "todos",
				Description:       "todos config",
				ChangeDescription: "",
				Content: utils.PairsToOrderedMap([]utils.Pair{
					{
						Key: "parameters",
						Value: *utils.PairsToOrderedMap([]utils.Pair{
							{
								Key: "api",
								Value: *utils.PairsToOrderedMap([]utils.Pair{
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
				BranchId:    111,
				ComponentId: "keboola.ex-generic",
				Id:          "456",
				ManifestPaths: model.ManifestPaths{
					Path:       "keboola.ex-generic/456-todos",
					ParentPath: "main",
				},
			},
		},
		{
			Local: &model.Config{
				BranchId:          123,
				ComponentId:       "keboola.ex-db-mysql",
				Id:                "896",
				Name:              "tables",
				Description:       "tables config",
				ChangeDescription: "",
				Content: utils.PairsToOrderedMap([]utils.Pair{
					{
						Key: "parameters",
						Value: *utils.PairsToOrderedMap([]utils.Pair{
							{
								Key: "db",
								Value: *utils.PairsToOrderedMap([]utils.Pair{
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
				BranchId:    123,
				ComponentId: "keboola.ex-db-mysql",
				Id:          "896",
				ManifestPaths: model.ManifestPaths{
					Path:       "keboola.ex-db-mysql/896-tables",
					ParentPath: "123-branch",
				},
			},
		},
		{
			Local: &model.Config{
				BranchId:          123,
				ComponentId:       "keboola.ex-generic",
				Id:                "456",
				Name:              "todos",
				Description:       "todos config",
				ChangeDescription: "",
				Content: utils.PairsToOrderedMap([]utils.Pair{
					{
						Key: "parameters",
						Value: *utils.PairsToOrderedMap([]utils.Pair{
							{
								Key: "api",
								Value: *utils.PairsToOrderedMap([]utils.Pair{
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
				BranchId:    123,
				ComponentId: "keboola.ex-generic",
				Id:          "456",
				ManifestPaths: model.ManifestPaths{
					Path:       "keboola.ex-generic/456-todos",
					ParentPath: "123-branch",
				},
			},
		},
	}
}

func complexLocalExpectedConfigRows() []*model.ConfigRowState {
	return []*model.ConfigRowState{
		{
			Local: &model.ConfigRow{
				BranchId:          123,
				ComponentId:       "keboola.ex-db-mysql",
				ConfigId:          "896",
				Id:                "56",
				Name:              "disabled",
				Description:       "",
				ChangeDescription: "",
				IsDisabled:        true,
				Content: utils.PairsToOrderedMap([]utils.Pair{
					{
						Key: "parameters",
						Value: *utils.PairsToOrderedMap([]utils.Pair{
							{Key: "incremental", Value: false},
						}),
					},
				}),
			},
			ConfigRowManifest: &model.ConfigRowManifest{
				Id:          "56",
				BranchId:    123,
				ComponentId: "keboola.ex-db-mysql",
				ConfigId:    "896",
				ManifestPaths: model.ManifestPaths{
					Path:       "56-disabled",
					ParentPath: "123-branch/keboola.ex-db-mysql/896-tables/rows",
				},
			},
		},
		{
			Local: &model.ConfigRow{
				BranchId:          123,
				ComponentId:       "keboola.ex-db-mysql",
				ConfigId:          "896",
				Id:                "34",
				Name:              "test_view",
				Description:       "row description",
				ChangeDescription: "",
				IsDisabled:        false,
				Content: utils.PairsToOrderedMap([]utils.Pair{
					{
						Key: "parameters",
						Value: *utils.PairsToOrderedMap([]utils.Pair{
							{Key: "incremental", Value: false},
						}),
					},
				}),
			},
			ConfigRowManifest: &model.ConfigRowManifest{
				Id:          "34",
				BranchId:    123,
				ComponentId: "keboola.ex-db-mysql",
				ConfigId:    "896",
				ManifestPaths: model.ManifestPaths{
					Path:       "34-test-view",
					ParentPath: "123-branch/keboola.ex-db-mysql/896-tables/rows",
				},
			},
		},
		{
			Local: &model.ConfigRow{
				BranchId:          123,
				ComponentId:       "keboola.ex-db-mysql",
				ConfigId:          "896",
				Id:                "12",
				Name:              "users",
				Description:       "",
				ChangeDescription: "",
				IsDisabled:        false,
				Content: utils.PairsToOrderedMap([]utils.Pair{
					{
						Key: "parameters",
						Value: *utils.PairsToOrderedMap([]utils.Pair{
							{Key: "incremental", Value: false},
						}),
					},
				}),
			},
			ConfigRowManifest: &model.ConfigRowManifest{
				Id:          "12",
				BranchId:    123,
				ComponentId: "keboola.ex-db-mysql",
				ConfigId:    "896",
				ManifestPaths: model.ManifestPaths{
					Path:       "12-users",
					ParentPath: "123-branch/keboola.ex-db-mysql/896-tables/rows",
				},
			},
		},
	}
}
