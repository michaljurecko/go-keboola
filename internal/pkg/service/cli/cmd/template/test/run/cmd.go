package run

import (
	"github.com/spf13/cobra"

	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/service/cli/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/service/cli/helpmsg"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/configmap"
	"github.com/keboola/keboola-as-code/internal/pkg/template"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
	testOp "github.com/keboola/keboola-as-code/pkg/lib/operation/template/local/test/run"
)

type Flags struct {
	TestName   string `configKey:"test-name" configUsage:"name of a single test to be run"`
	LocalOnly  bool   `configKey:"local-only" configUsage:"run a local test only"`
	RemoteOnly bool   `configKey:"remote-only" configUsage:"run a remote test only"`
	Verbose    bool   `configKey:"verbose" configUsage:"show details about running tests"`
}

func DefaultFlags() Flags {
	return Flags{}
}

func Command(p dependencies.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [template] [version]",
		Short: helpmsg.Read(`template/test/run/short`),
		Long:  helpmsg.Read(`template/test/run/long`),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := Flags{}
			if err := p.BaseScope().ConfigBinder().Bind(cmd.Flags(), args, &f); err != nil {
				return err
			}

			// Options
			options := testOp.Options{
				LocalOnly:  f.LocalOnly,
				RemoteOnly: f.RemoteOnly,
				TestName:   f.TestName,
				Verbose:    f.Verbose,
			}

			// Get dependencies
			d, err := p.LocalCommandScope(cmd.Context(), dependencies.WithDefaultStorageAPIHost())
			if err != nil {
				return err
			}

			// Get template repository
			repo, _, err := d.LocalTemplateRepository(cmd.Context())
			if err != nil {
				return err
			}

			// Load templates
			templates := make([]*template.Template, 0)
			if len(args) >= 1 {
				// Optional version argument
				var versionArg string
				if len(args) > 1 {
					versionArg = args[1]
				}
				tmpl, err := d.Template(cmd.Context(), model.NewTemplateRef(repo.Definition(), args[0], versionArg))
				if err != nil {
					return errors.Errorf(`loading test for template "%s" failed: %w`, args[0], err)
				}
				templates = append(templates, tmpl)
			} else {
				for _, t := range repo.Templates() {
					v, err := t.DefaultVersionOrErr()
					if err != nil {
						return errors.Errorf(`loading default version for template "%s" failed: %w`, t.ID, err)
					}
					tmpl, err := d.Template(cmd.Context(), model.NewTemplateRef(repo.Definition(), t.ID, v.Version.String()))
					if err != nil {
						return errors.Errorf(`loading test for template "%s" failed: %w`, t.ID, err)
					}
					templates = append(templates, tmpl)
				}
			}

			// Test templates
			errs := errors.NewMultiError()
			for _, tmpl := range templates {
				err := testOp.Run(cmd.Context(), tmpl, options, d)
				if err != nil {
					errs.Append(err)
				}
			}
			return errs.ErrorOrNil()
		},
	}

	configmap.MustGenerateFlags(cmd.Flags(), DefaultFlags())

	return cmd
}
