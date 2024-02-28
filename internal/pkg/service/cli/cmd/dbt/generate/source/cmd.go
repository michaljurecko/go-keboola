package source

import (
	"github.com/spf13/cobra"

	"github.com/keboola/keboola-as-code/internal/pkg/service/cli/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/service/cli/helpmsg"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/configmap"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
	"github.com/keboola/keboola-as-code/pkg/lib/operation/dbt/generate/sources"
)

type Flags struct {
	StorageAPIHost configmap.Value[string] `configKey:"storage-api-host" configShorthand:"H" configUsage:"storage API host, eg. \"connection.keboola.com\""`
	TargetName     configmap.Value[string] `configKey:"target-name" configShorthand:"T" configUsage:"target name of the profile"`
}

func SourcesCommand(p dependencies.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   `sources`,
		Short: helpmsg.Read(`dbt/generate/sources/short`),
		Long:  helpmsg.Read(`dbt/generate/sources/long`),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check that we are in dbt directory
			if _, _, err := p.LocalDbtProject(cmd.Context()); err != nil {
				return err
			}

			// Get dependencies
			d, err := p.RemoteCommandScope(cmd.Context())
			if err != nil {
				return err
			}

			f := Flags{}
			if err = p.BaseScope().ConfigBinder().Bind(cmd.Flags(), args, &f); err != nil {
				return err
			}

			// Get default branch
			branch, err := d.KeboolaProjectAPI().GetDefaultBranchRequest().Send(cmd.Context())
			if err != nil {
				return errors.Errorf("cannot get default branch: %w", err)
			}

			// Ask options
			targetName, err := p.BaseScope().Dialogs().AskTargetName(f.TargetName)
			if err != nil {
				return err
			}

			return sources.Run(cmd.Context(), sources.Options{BranchKey: branch.BranchKey, TargetName: targetName}, d)
		},
	}

	configmap.MustGenerateFlags(cmd.Flags(), Flags{})

	return cmd
}
