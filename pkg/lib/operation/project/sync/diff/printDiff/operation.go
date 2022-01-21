package printDiff

import (
	"github.com/keboola/keboola-as-code/internal/pkg/diff"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/project"
	createDiff "github.com/keboola/keboola-as-code/pkg/lib/operation/project/sync/diff/create"
	loadState "github.com/keboola/keboola-as-code/pkg/lib/operation/state/load"
)

type Options struct {
	PrintDetails      bool
	LogUntrackedPaths bool
}

type dependencies interface {
	Logger() log.Logger
	ProjectState(loadOptions loadState.Options) (*project.State, error)
}

func LoadStateOptions() loadState.Options {
	return loadState.Options{
		LoadLocalState:          true,
		LoadRemoteState:         true,
		IgnoreNotFoundErr:       false,
		IgnoreInvalidLocalState: false,
	}
}

func Run(o Options, d dependencies) (*diff.Results, error) {
	logger := d.Logger()

	// Load state
	projectState, err := d.ProjectState(LoadStateOptions())
	if err != nil {
		return nil, err
	}

	// Diff
	results, err := createDiff.Run(createDiff.Options{Objects: projectState})
	if err != nil {
		return nil, err
	}

	// Log untracked paths
	if o.LogUntrackedPaths {
		projectState.LogUntrackedPaths(logger)
	}

	if results.Equal {
		logger.Info("No difference.")
	} else {
		// Explain
		logger.Info(diff.ChangeMark + " changed")
		logger.Info(diff.OnlyInRemoteMark + " remote state")
		logger.Info(diff.OnlyInLocalMark + " local state")
		logger.Info("")

		// Print diff
		logger.Info("Diff:")
		for _, line := range results.Format(o.PrintDetails) {
			logger.Info(line)
		}
	}

	return results, nil
}
