package cli

import (
	"github.com/spf13/cobra"

	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/manifest"
	"github.com/keboola/keboola-as-code/internal/pkg/plan"
	"github.com/keboola/keboola-as-code/internal/pkg/state"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
)

const (
	persistShortDescription = `Persist created and deleted configs/rows in manifest`
	persistLongDescription  = `Command "persist"

This command writes the changes from the filesystem to the manifest.
- If you have created a new config/row, this command will write record to the manifest with a unique ID.
- If you have deleted a config/row, this command will delete record from the manifest.

No changes are made to the remote state of the project.

If you also want to change the remote state,
call the "push" command after the "persist" command.
`
)

func persistCommand(root *rootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "persist",
		Short: persistShortDescription,
		Long:  persistLongDescription,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			logger := root.logger

			// Validate project directory
			if err := ValidateMetadataFound(root.fs); err != nil {
				return err
			}

			// Validate token
			root.options.AskUser(root.prompt, "ApiToken")
			if err := root.ValidateOptions([]string{"ApiToken"}); err != nil {
				return err
			}

			// Load manifest
			projectManifest, err := manifest.LoadManifest(root.fs)
			if err != nil {
				return err
			}

			// Validate token and get API
			root.options.ApiHost = projectManifest.Project.ApiHost
			api, err := root.GetStorageApi()
			if err != nil {
				return err
			}

			// Get Scheduler API
			schedulerApi, err := root.GetSchedulerApi()
			if err != nil {
				return err
			}

			// Load project local state
			stateOptions := state.NewOptions(projectManifest, api, schedulerApi, root.ctx, logger)
			stateOptions.LoadLocalState = true
			stateOptions.SkipNotFoundErr = true
			projectState, ok := state.LoadState(stateOptions)
			if ok {
				logger.Debugf("Project local state has been successfully loaded.")
			} else if projectState.LocalErrors().Len() > 0 {
				return utils.PrefixError("project local state is invalid", projectState.LocalErrors())
			}

			// Get plan
			persist, err := plan.Persist(projectState)
			if err != nil {
				return err
			}

			// Log plan
			persist.Log(log.ToInfoWriter(logger))

			// Dry run?
			dryRun := root.options.GetBool("dry-run")
			if dryRun {
				logger.Info("Dry run, nothing changed.")
				logger.Info(`Persist done.`)
				return nil
			}

			// Invoke
			if err := persist.Invoke(logger, api, projectState); err != nil {
				return utils.PrefixError(`cannot persist objects`, err)
			}
			logger.Info(`Persist done.`)

			// Print remaining untracked paths
			projectState.LogUntrackedPaths(root.logger)

			// Normalize paths
			if err := Rename(root.ctx, projectState, logger, false, false); err != nil {
				return err
			}

			// Save manifest
			if _, err := SaveManifest(projectManifest, logger); err != nil {
				return err
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().Bool("dry-run", false, "print what needs to be done")

	return cmd
}
