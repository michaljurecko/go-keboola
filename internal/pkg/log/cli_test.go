// nolint: forbidigo
package log

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/keboola/go-utils/pkg/wildcards"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"

	"github.com/keboola/keboola-as-code/internal/pkg/service/common/ctxattr"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/ioutil"
)

func TestCliLogger_New(t *testing.T) {
	t.Parallel()
	stdout := ioutil.NewAtomicWriter()
	stderr := ioutil.NewAtomicWriter()
	logger := NewCliLogger(stdout, stderr, nil, LogFormatConsole, false)
	assert.NotNil(t, logger)
}

func TestCliLogger_File(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "log-file.txt")
	file, err := NewLogFile(filePath)
	assert.NoError(t, err)

	stdout := ioutil.NewAtomicWriter()
	stderr := ioutil.NewAtomicWriter()
	logger := NewCliLogger(stdout, stderr, file, LogFormatConsole, false)

	logger.DebugCtx(context.Background(), "Debug msg")
	logger.InfoCtx(context.Background(), "Info msg")
	logger.WarnCtx(context.Background(), "Warn msg")
	logger.ErrorCtx(context.Background(), "Error msg")
	assert.NoError(t, file.File().Close())

	// Assert, all levels logged with the level prefix
	expected := `
{"level":"debug","time":"%s","message":"Debug msg"}
{"level":"info","time":"%s","message":"Info msg"}
{"level":"warn","time":"%s","message":"Warn msg"}
{"level":"error","time":"%s","message":"Error msg"}
`

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	wildcards.Assert(t, expected, string(content))
}

func TestCliLogger_VerboseFalse(t *testing.T) {
	t.Parallel()
	stdout := ioutil.NewAtomicWriter()
	stderr := ioutil.NewAtomicWriter()
	logger := NewCliLogger(stdout, stderr, nil, LogFormatConsole, false)
	// Check that context attributes don't appear in stdout/stderr.
	ctx := ctxattr.ContextWith(context.Background(), attribute.String("extra", "value"))

	logger.DebugCtx(ctx, "Debug msg")
	logger.InfoCtx(ctx, "Info msg")
	logger.WarnCtx(ctx, "Warn msg")
	logger.ErrorCtx(ctx, "Error msg")

	// Assert
	// info      -> stdout
	// warn, err -> stderr
	expectedOut := "Info msg\n"
	expectedErr := "Warn msg\nError msg\n"
	assert.Equal(t, expectedOut, stdout.String())
	assert.Equal(t, expectedErr, stderr.String())
}

func TestCliLogger_VerboseTrue(t *testing.T) {
	t.Parallel()
	stdout := ioutil.NewAtomicWriter()
	stderr := ioutil.NewAtomicWriter()
	logger := NewCliLogger(stdout, stderr, nil, LogFormatConsole, true)
	// Check that context attributes don't appear in stdout/stderr.
	ctx := ctxattr.ContextWith(context.Background(), attribute.String("extra", "value"))

	logger.DebugCtx(ctx, "Debug msg")
	logger.InfoCtx(ctx, "Info msg")
	logger.WarnCtx(ctx, "Warn msg")
	logger.ErrorCtx(ctx, "Error msg")

	// Assert
	// debug (verbose), info -> stdout
	// warn, err             -> stderr
	expectedOut := "DEBUG\tDebug msg\nINFO\tInfo msg\n"
	expectedErr := "WARN\tWarn msg\nERROR\tError msg\n"
	assert.Equal(t, expectedOut, stdout.String())
	assert.Equal(t, expectedErr, stderr.String())
}

func TestCliLogger_JSONVerboseFalse(t *testing.T) {
	t.Parallel()
	stdout := ioutil.NewAtomicWriter()
	stderr := ioutil.NewAtomicWriter()
	logger := NewCliLogger(stdout, stderr, nil, LogFormatJSON, false)
	ctx := context.Background()

	logger.DebugCtx(ctx, "Debug msg")
	logger.InfoCtx(ctx, "Info msg")
	logger.WarnCtx(ctx, "Warn msg")
	logger.ErrorCtx(ctx, "Error msg")

	// Assert
	// info      -> stdout
	// warn, err -> stderr
	expectedOut := `
{"level":"info","time":"%s","message":"Info msg"}
`
	expectedErr := `
{"level":"warn","time":"%s","message":"Warn msg"}
{"level":"error","time":"%s","message":"Error msg"}
`

	wildcards.Assert(t, expectedOut, stdout.String())
	wildcards.Assert(t, expectedErr, stderr.String())
}

func TestCliLogger_JSONVerboseTrue(t *testing.T) {
	t.Parallel()
	stdout := ioutil.NewAtomicWriter()
	stderr := ioutil.NewAtomicWriter()
	logger := NewCliLogger(stdout, stderr, nil, LogFormatJSON, true)
	ctx := context.Background()

	logger.DebugCtx(ctx, "Debug msg")
	logger.InfoCtx(ctx, "Info msg")
	logger.WarnCtx(ctx, "Warn msg")
	logger.ErrorCtx(ctx, "Error msg")

	// Assert
	// debug (verbose), info -> stdout
	// warn, err             -> stderr
	expectedOut := `
{"level":"debug","time":"%s","message":"Debug msg"}
{"level":"info","time":"%s","message":"Info msg"}
`
	expectedErr := `
{"level":"warn","time":"%s","message":"Warn msg"}
{"level":"error","time":"%s","message":"Error msg"}
`

	wildcards.Assert(t, expectedOut, stdout.String())
	wildcards.Assert(t, expectedErr, stderr.String())
}

func TestCliLogger_AttributeReplace(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "log-file.txt")
	file, err := NewLogFile(filePath)
	assert.NoError(t, err)

	stdout := ioutil.NewAtomicWriter()
	stderr := ioutil.NewAtomicWriter()
	logger := NewCliLogger(stdout, stderr, file, LogFormatConsole, true)

	ctx := ctxattr.ContextWith(context.Background(), attribute.String("extra", "value"), attribute.Int("count", 4))

	logger.Debug(ctx, "Debug msg <extra> (<count>)")
	logger.Info(ctx, "Info msg <extra> (<count>)")
	logger.Warn(ctx, "Warn msg <extra> (<count>)")
	logger.Error(ctx, "Error msg <extra> (<count>)")
	assert.NoError(t, file.File().Close())

	// Assert, all levels logged with the level prefix
	expected := `
{"level":"debug","time":"%s","message":"Debug msg value (4)","count":4,"extra":"value"}
{"level":"info","time":"%s","message":"Info msg value (4)","count":4,"extra":"value"}
{"level":"warn","time":"%s","message":"Warn msg value (4)","count":4,"extra":"value"}
{"level":"error","time":"%s","message":"Error msg value (4)","count":4,"extra":"value"}
`

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	wildcards.Assert(t, expected, string(content))

	expectedOut := "DEBUG\tDebug msg value (4)\nINFO\tInfo msg value (4)\n"
	expectedErr := "WARN\tWarn msg value (4)\nERROR\tError msg value (4)\n"
	assert.Equal(t, expectedOut, stdout.String())
	assert.Equal(t, expectedErr, stderr.String())
}

func TestCliLogger_WithComponent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "log-file.txt")
	file, err := NewLogFile(filePath)
	assert.NoError(t, err)

	stdout := ioutil.NewAtomicWriter()
	stderr := ioutil.NewAtomicWriter()
	logger := NewCliLogger(stdout, stderr, file, LogFormatConsole, true)

	logger = logger.WithComponent("component").WithComponent("subcomponent")

	ctx := context.Background()

	logger.Debug(ctx, "Debug msg")
	logger.Info(ctx, "Info msg")
	logger.Warn(ctx, "Warn msg")
	logger.Error(ctx, "Error msg")
	assert.NoError(t, file.File().Close())

	// Assert, all levels logged with the level prefix
	expected := `
{"level":"debug","time":"%s","message":"Debug msg","component":"component.subcomponent"}
{"level":"info","time":"%s","message":"Info msg","component":"component.subcomponent"}
{"level":"warn","time":"%s","message":"Warn msg","component":"component.subcomponent"}
{"level":"error","time":"%s","message":"Error msg","component":"component.subcomponent"}
`

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	wildcards.Assert(t, expected, string(content))

	expectedOut := "DEBUG\tDebug msg\nINFO\tInfo msg\n"
	expectedErr := "WARN\tWarn msg\nERROR\tError msg\n"
	assert.Equal(t, expectedOut, stdout.String())
	assert.Equal(t, expectedErr, stderr.String())
}
