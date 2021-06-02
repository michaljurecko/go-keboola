package tests

import (
	"bytes"
	"github.com/google/shlex"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"keboola-as-code/src/utils"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestFunctional runs one functional test per each sub-directory
func TestFunctional(t *testing.T) {
	_, testFile, _, _ := runtime.Caller(0)
	rootDir := filepath.Dir(testFile)

	// Create temp dir
	tempDir := t.TempDir()

	// Compile binary, it will be run in the tests
	projectDir := rootDir + "/../.."
	binary := CompileBinary(t, projectDir, tempDir)

	// Run binary in each data dir
	for _, testDir := range GetTestDirs(t, rootDir) {
		// Run test for each directory
		t.Run(filepath.Base(testDir), func(t *testing.T) {
			RunFunctionalTest(t, testDir, binary)
		})
	}
}

// RunFunctionalTest runs one functional test.
func RunFunctionalTest(t *testing.T, testDir string, binary string) {
	// Create runtime dir
	workingDir := t.TempDir()

	// Copy all from in dir to runtime dir
	inDir := testDir + "/in"
	if !utils.FileExists(inDir) {
		t.Fatalf("Missing directory \"%s\".", inDir)
	}
	err := copy.Copy(inDir, workingDir)
	if err != nil {
		t.Fatalf("Copy error: %s", err)
	}

	// Load command arguments from file
	argsFile := testDir + "/args"
	argsStr := strings.TrimSpace(utils.GetFileContent(t, argsFile))
	args, err := shlex.Split(argsStr)
	if err != nil {
		t.Fatalf("Cannot parse args \"%s\": %s", argsStr, err)
	}

	// Prepare command
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, args...)
	cmd.Dir = workingDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			t.Fatalf("Command failed: %s", err)
		}
	}

	AssertExpectations(t, testDir, workingDir, exitCode, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
}

// CompileBinary compiles component to binary used in this test
func CompileBinary(t *testing.T, projectDir string, tempDir string) string {
	var stdout, stderr bytes.Buffer
	binaryPath := tempDir + "/bin_func_tests"
	cmd := exec.Command(projectDir+"/scripts/compile.sh", binaryPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		t.Fatalf("Compilation failed: %s\n%s\n%s\n", err, stdout.Bytes(), stderr.Bytes())
	}

	return binaryPath
}

// GetTestDirs returns list of all dirs in the root directory.
func GetTestDirs(t *testing.T, root string) []string {
	var dirs []string

	// Iterate over directory structure
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Stop on error
		if err != nil {
			return err
		}

		// Ignore root
		if path == root {
			return nil
		}

		// Skip sub-directories
		if info.IsDir() {
			dirs = append(dirs, path)
			return fs.SkipDir
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	return dirs
}

// AssertExpectations compares expectations with the actual state.
func AssertExpectations(
	t *testing.T,
	testDir string,
	workingDir string,
	exitCode int,
	stdout string,
	stderr string,
) {
	expectedDir := testDir + "/out"
	if !utils.FileExists(expectedDir) {
		t.Fatalf("Missing directory \"%s\".", expectedDir)
	}

	expectedStdout := utils.GetFileContent(t, testDir+"/expected-stdout")
	expectedStderr := utils.GetFileContent(t, testDir+"/expected-stderr")

	// Assert exit code
	expectedCodeStr := utils.GetFileContent(t, testDir+"/expected-code")
	expectedCode, _ := strconv.ParseInt(strings.TrimSpace(expectedCodeStr), 10, 32)
	assert.Equal(
		t,
		int(expectedCode),
		exitCode,
		"Unexpected exit code.\nSTDOUT:\n%s\n\nSTDERR:\n%s\n\n",
		stdout,
		stderr,
	)

	// Assert STDOUT and STDERR
	utils.AssertWildcards(t, expectedStdout, stdout, "Unexpected STDOUT.")
	utils.AssertWildcards(t, expectedStderr, stderr, "Unexpected STDERR.")

	// Compare actual and expected dirs
	utils.AssertDirectoryContentsSame(t, expectedDir, workingDir)
}
