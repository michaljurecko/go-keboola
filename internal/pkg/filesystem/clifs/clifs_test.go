// nolint: forbidigo
package clifs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/keboola/keboola-as-code/internal/pkg/dbt"
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
)

func TestFindProjectDir(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	metadataDir := filepath.Join(projectDir, filesystem.MetadataDir)
	assert.NoError(t, os.MkdirAll(metadataDir, 0o755))

	// Working dir is a sub-dir of the project dir
	workingDir := filepath.Join(projectDir, `foo`, `bar`)
	assert.NoError(t, os.MkdirAll(workingDir, 0o755))

	dir, err := find(log.NewNopLogger(), workingDir)
	assert.NoError(t, err)
	assert.Equal(t, projectDir, dir)
}

func TestFindDbtDir(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	dbtProjectFile := filepath.Join(projectDir, dbt.ProjectFile)
	assert.NoError(t, os.WriteFile(dbtProjectFile, []byte("\n"), 0o700))

	// Working dir is a sub-dir of the dbt project dir
	workingDir := filepath.Join(projectDir, `foo`, `bar`)
	assert.NoError(t, os.MkdirAll(workingDir, 0o755))

	dir, err := find(log.NewNopLogger(), workingDir)
	assert.NoError(t, err)
	assert.Equal(t, projectDir, dir)
}

func TestFindNothingFound(t *testing.T) {
	t.Parallel()
	workingDir := t.TempDir()
	dir, err := find(log.NewNopLogger(), workingDir)
	assert.NoError(t, err)
	assert.Equal(t, workingDir, dir)
}
