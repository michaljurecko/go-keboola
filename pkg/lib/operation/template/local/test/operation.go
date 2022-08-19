package test

import (
	"context"
	"fmt"
	"github.com/keboola/go-client/pkg/storageapi"
	"github.com/keboola/go-utils/pkg/deepcopy"
	cliDeps "github.com/keboola/keboola-as-code/internal/pkg/cli/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/cli/dialog"
	"github.com/keboola/keboola-as-code/internal/pkg/cli/options"
	"github.com/keboola/keboola-as-code/internal/pkg/cli/prompt/nop"
	"github.com/keboola/keboola-as-code/internal/pkg/env"
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem/aferofs"
	fixtures "github.com/keboola/keboola-as-code/internal/pkg/fixtures/local"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/project"
	"github.com/keboola/keboola-as-code/internal/pkg/template"
	"github.com/keboola/keboola-as-code/internal/pkg/template/input"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/testfs"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/testhelper"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/testhelper/storageenvmock"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/testproject"
	useTemplate "github.com/keboola/keboola-as-code/pkg/lib/operation/project/local/template/use"
	syncPush "github.com/keboola/keboola-as-code/pkg/lib/operation/project/sync/push"
	loadState "github.com/keboola/keboola-as-code/pkg/lib/operation/state/load"
	"os"
	"strconv"
)

type Options struct {
	LocalOnly  bool   // run local tests only
	RemoteOnly bool   // run remote tests only
	TestName   string // run only selected test
}

type dependencies interface {
	Ctx() context.Context
	Logger() log.Logger
}

func Run(tmpl *template.Template, o Options, d dependencies) (err error) {
	tempDir, err := os.MkdirTemp("", "kac-test-template-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil { // nolint: forbidigo
			d.Logger().Warnf(`cannot remove temp dir "%s": %w`, tempDir, err)
		}
	}()

	repoDirFS, err := prepareRepoFS(tempDir, tmpl)

	// Run through all tests
	testsList, err := tmpl.ListTests()
	if err != nil {
		return err
	}

	for _, testName := range testsList {
		// Run only a single test?
		if o.TestName != "" && o.TestName != testName {
			continue
		}

		if !o.RemoteOnly {
			d.Logger().Infof(`Running local test "%s".`, testName)
			if err := runLocalTest(testName, tmpl, repoDirFS, d); err != nil {
				return fmt.Errorf(`running local test "%s" for template "%s" failed: %w`, testName, tmpl.TemplateId(), err)
			}
			d.Logger().Infof(`Local test "%s" finished.`, testName)
		}

		if !o.LocalOnly {
			d.Logger().Infof(`Running remote test "%s".`, testName)
			if err := runRemoteTest(testName, tmpl, repoDirFS, d); err != nil {
				return fmt.Errorf(`running remote test "%s" for template "%s" failed: %w`, testName, tmpl.TemplateId(), err)
			}
			d.Logger().Infof(`Remote test "%s" finished.`, testName)

		}
	}

	return nil
}

func runLocalTest(testName string, tmpl *template.Template, repoFS filesystem.Fs, d dependencies) error {
	// Get a test project
	envs, err := env.FromOs()
	if err != nil {
		return err
	}
	testPrj, unlockFn, err := testproject.GetTestProject(envs)
	if err != nil {
		return err
	}
	defer unlockFn()

	branchID := 1

	// Load fixture with minimal project
	fixPrjEnvs := env.Empty()
	fixPrjEnvs.Set("TEST_KBC_STORAGE_API_HOST", testPrj.StorageAPIHost())
	fixPrjEnvs.Set("LOCAL_PROJECT_ID", strconv.Itoa(testPrj.ID()))
	fixPrjEnvs.Set("LOCAL_STATE_MAIN_BRANCH_ID", strconv.Itoa(branchID))
	projectFS, err := fixtures.LoadFS("empty-branch", fixPrjEnvs)
	if err != nil {
		return err
	}

	opts := options.New()
	opts.Set(`storage-api-host`, testPrj.StorageAPIHost())
	opts.Set(`storage-api-token`, testPrj.StorageAPIToken().Token)
	tmplDeps := cliDeps.NewContainer(d.Ctx(), env.Empty(), repoFS, dialog.New(nop.New()), d.Logger(), opts)
	prjDeps := cliDeps.NewContainer(d.Ctx(), env.Empty(), projectFS, dialog.New(nop.New()), d.Logger(), opts)

	// Re-init template with set-up Storage client
	tmpl, err = tmplDeps.Template(tmpl.Reference())
	if err != nil {
		return err
	}

	// Load project state
	prj, err := project.New(projectFS, true, prjDeps)
	if err != nil {
		return err
	}
	prjState, err := prj.LoadState(loadState.LocalOperationOptions())
	if err != nil {
		return err
	}
	d.Logger().Debugf(`Working directory set up.`)

	// Read inputs
	inputsFile, err := tmpl.TestInputs(testName)
	if err != nil {
		return err
	}

	inputValues := make(template.InputsValues, 0)
	err = tmpl.Inputs().ToExtended().VisitInputs(func(group *input.StepsGroupExt, step *input.StepExt, inputDef *input.Input) error {
		var inputValue template.InputValue
		if v, found := inputsFile[inputDef.Id]; found {
			inputValue, err = template.ParseInputValue(v, inputDef, true)
			if err != nil {
				return utils.PrefixError(err.Error(), fmt.Errorf("please fix the value in the inputs JSON file"))
			}
		} else {
			inputValue, err = template.ParseInputValue(inputDef.DefaultOrEmpty(), inputDef, true)
			if err != nil {
				return utils.PrefixError(err.Error(), fmt.Errorf("please define value in the inputs JSON file"))
			}
		}
		inputValues = append(inputValues, inputValue)
		return nil
	})
	if err != nil {
		return err
	}
	d.Logger().Debugf(`Inputs prepared.`)

	// Use template
	tmplOpts := useTemplate.Options{
		InstanceName: "test",
		TargetBranch: model.BranchKey{Id: storageapi.BranchID(branchID)},
		Inputs:       inputValues,
	}
	_, _, err = useTemplate.Run(prjState, tmpl, tmplOpts, prjDeps)

	// Copy expected state and replace ENVs
	expectedDirFs, err := tmpl.TestExpectedOutFS(testName)
	if err != nil {
		return err
	}
	replaceEnvs := env.Empty()
	replaceEnvs.Set("STORAGE_API_HOST", testPrj.StorageAPIHost())
	replaceEnvs.Set("PROJECT_ID", strconv.Itoa(testPrj.ID()))
	replaceEnvs.Set("MAIN_BRANCH_ID", strconv.Itoa(branchID))
	envProvider := storageenvmock.CreateStorageEnvMockTicketProvider(d.Ctx(), replaceEnvs)
	testhelper.ReplaceEnvsDir(projectFS, `/`, envProvider)
	testhelper.ReplaceEnvsDirWithSeparator(expectedDirFs, `/`, envProvider, "__")

	// Compare actual and expected dirs
	return testhelper.DirectoryContentsSame(expectedDirFs, `/`, projectFS, `/`)
}

func runRemoteTest(testName string, tmpl *template.Template, repoFS filesystem.Fs, d dependencies) error {
	// Get a test project
	envs, err := env.FromOs()
	if err != nil {
		return err
	}
	testPrj, unlockFn, err := testproject.GetTestProject(envs)
	if err != nil {
		return err
	}
	defer unlockFn()

	defBranch, err := testPrj.DefaultBranch()
	if err != nil {
		return err
	}
	branchID := int(defBranch.ID)

	// Load fixture with minimal project
	fixPrjEnvs := env.Empty()
	fixPrjEnvs.Set("TEST_KBC_STORAGE_API_HOST", testPrj.StorageAPIHost())
	fixPrjEnvs.Set("LOCAL_PROJECT_ID", strconv.Itoa(testPrj.ID()))
	fixPrjEnvs.Set("LOCAL_STATE_MAIN_BRANCH_ID", strconv.Itoa(branchID))
	prjFS, err := fixtures.LoadFS("empty-branch", fixPrjEnvs)
	if err != nil {
		return err
	}

	opts := options.New()
	opts.Set(`storage-api-host`, testPrj.StorageAPIHost())
	opts.Set(`storage-api-token`, testPrj.StorageAPIToken().Token)
	tmplDeps := cliDeps.NewContainer(d.Ctx(), env.Empty(), repoFS, dialog.New(nop.New()), d.Logger(), opts)
	projectDeps := cliDeps.NewContainer(d.Ctx(), env.Empty(), prjFS, dialog.New(nop.New()), d.Logger(), opts)

	// Re-init template with set-up Storage client
	tmpl, err = tmplDeps.Template(tmpl.Reference())
	if err != nil {
		return err
	}

	// Load project state
	prj, err := project.New(prjFS, true, projectDeps)
	// Create fake manifest
	m := project.NewManifest(testPrj.ID(), testPrj.StorageAPIHost())
	// Load only target branch
	m.Filter().SetAllowedKeys([]model.Key{model.BranchKey{Id: storageapi.BranchID(branchID)}})

	if err != nil {
		return err
	}
	prjState, err := prj.LoadState(loadState.Options{LoadRemoteState: true})
	if err != nil {
		return err
	}
	d.Logger().Debugf(`Working directory set up.`)

	// Read inputs
	inputsFile, err := tmpl.TestInputs(testName)
	if err != nil {
		return err
	}

	inputValues := make(template.InputsValues, 0)
	err = tmpl.Inputs().ToExtended().VisitInputs(func(group *input.StepsGroupExt, step *input.StepExt, inputDef *input.Input) error {
		var inputValue template.InputValue
		if v, found := inputsFile[inputDef.Id]; found {
			inputValue, err = template.ParseInputValue(v, inputDef, true)
			if err != nil {
				return utils.PrefixError(err.Error(), fmt.Errorf("please fix the value in the inputs JSON file"))
			}
		} else {
			inputValue, err = template.ParseInputValue(inputDef.DefaultOrEmpty(), inputDef, true)
			if err != nil {
				return utils.PrefixError(err.Error(), fmt.Errorf("please define value in the inputs JSON file"))
			}
		}
		inputValues = append(inputValues, inputValue)
		return nil
	})
	if err != nil {
		return err
	}
	d.Logger().Debugf(`Inputs prepared.`)

	// Copy remote state to the local
	prjState, err = prj.LoadState(loadState.InitOptions(true))
	if err != nil {
		return err
	}
	for _, objectState := range prjState.All() {
		objectState.SetLocalState(deepcopy.Copy(objectState.RemoteState()).(model.Object))
	}

	// Use template
	tmplOpts := useTemplate.Options{
		InstanceName: "test",
		TargetBranch: model.BranchKey{Id: storageapi.BranchID(branchID)},
		Inputs:       inputValues,
	}
	_, _, err = useTemplate.Run(prjState, tmpl, tmplOpts, projectDeps)

	// Copy expected state and replace ENVs
	expectedDirFs, err := tmpl.TestExpectedOutFS(testName)
	if err != nil {
		return err
	}
	replaceEnvs := env.Empty()
	replaceEnvs.Set("STORAGE_API_HOST", testPrj.StorageAPIHost())
	replaceEnvs.Set("PROJECT_ID", strconv.Itoa(testPrj.ID()))
	replaceEnvs.Set("MAIN_BRANCH_ID", strconv.Itoa(branchID))
	envProvider := storageenvmock.CreateStorageEnvMockTicketProvider(d.Ctx(), replaceEnvs)
	testhelper.ReplaceEnvsDir(prjFS, `/`, envProvider)
	testhelper.ReplaceEnvsDirWithSeparator(expectedDirFs, `/`, envProvider, "__")

	// Compare actual and expected dirs
	err = testhelper.DirectoryContentsSame(expectedDirFs, `/`, prjFS, `/`)
	if err != nil {
		return err
	}

	// E2E test
	// Push the project
	pushOpts := syncPush.Options{
		Encrypt:           true,
		DryRun:            false,
		SkipValidation:    false,
		AllowRemoteDelete: true,
		LogUntrackedPaths: true,
		ChangeDescription: "",
	}
	return syncPush.Run(prjState, pushOpts, projectDeps)
}

func prepareRepoFS(tempDir string, tmpl *template.Template) (filesystem.Fs, error) {
	// Create virtual fs for working dir
	repoFS := testfs.NewBasePathLocalFs(tempDir)

	// Load fixture with minimal repository
	fixRepoEnvs := env.Empty()
	fixRepoEnvs.Set("TEMPLATE_ID", tmpl.TemplateId())
	fixRepoEnvs.Set("TEMPLATE_NAME", tmpl.FullName())
	fixRepoEnvs.Set("TEMPLATE_VERSION", tmpl.Version())
	fixRepoFS, err := fixtures.LoadFS("repository-basic", fixRepoEnvs)
	if err != nil {
		return nil, err
	}
	if err := aferofs.CopyFs2Fs(fixRepoFS, `/`, repoFS, `/`); err != nil {
		return nil, err
	}

	// Load the template dir
	if err := aferofs.CopyFs2Fs(tmpl.SrcDir(), `/`, repoFS, fmt.Sprintf(`/%s/%s/src`, tmpl.TemplateId(), tmpl.Version())); err != nil {
		return nil, err
	}
	testsDir, err := tmpl.TestsDir()
	if err != nil {
		return nil, err
	}
	if err := aferofs.CopyFs2Fs(testsDir, `/`, repoFS, fmt.Sprintf(`/%s/%s/tests`, tmpl.TemplateId(), tmpl.Version())); err != nil {
		return nil, err
	}

	return repoFS, nil
}
