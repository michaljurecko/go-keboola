package test

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/keboola/keboola-as-code/internal/pkg/cli/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/cli/helpmsg"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/template"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
	testOp "github.com/keboola/keboola-as-code/pkg/lib/operation/template/local/test/run"
)

func RunCommand(p dependencies.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [template] [version]",
		Short: helpmsg.Read(`template/test/run/short`),
		Long:  helpmsg.Read(`template/test/run/long`),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := p.DependenciesForLocalCommand()
			if err != nil {
				return err
			}

			// Get template repository
			repo, _, err := d.LocalTemplateRepository(d.CommandCtx())
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
				tmpl, err := d.Template(d.CommandCtx(), model.NewTemplateRef(repo.Definition(), args[0], versionArg))
				if err != nil {
					return fmt.Errorf(`loading test for template "%s" failed: %w`, args[0], err)
				}
				templates = append(templates, tmpl)
			} else {
				for _, t := range repo.Templates() {
					v, err := t.DefaultVersionOrErr()
					if err != nil {
						return fmt.Errorf(`loading default version for template "%s" failed: %w`, t.Id, err)
					}
					tmpl, err := d.Template(d.CommandCtx(), model.NewTemplateRef(repo.Definition(), t.Id, v.Version.String()))
					if err != nil {
						return fmt.Errorf(`loading test for template "%s" failed: %w`, t.Id, err)
					}
					templates = append(templates, tmpl)
				}
			}

			// Options
			options := testOp.Options{
				LocalOnly:  d.Options().GetBool("local-only"),
				RemoteOnly: d.Options().GetBool("remote-only"),
				TestName:   d.Options().GetString("test-name"),
				Verbose:    d.Options().GetBool("verbose"),
			}

			// Test templates
			errs := utils.NewMultiError()
			for _, tmpl := range templates {
				err := testOp.Run(d.CommandCtx(), tmpl, options, d)
				if err != nil {
					errs.Append(err)
				}
			}
			return errs.ErrorOrNil()
		},
	}

	cmd.Flags().SortFlags = true
	cmd.Flags().String("test-name", "", "name of a single test to be run")
	cmd.Flags().Bool("local-only", false, "run a local test only")
	cmd.Flags().Bool("remote-only", false, "run a remote test only")
	cmd.Flags().Bool("verbose", false, "show details about running tests")

	return cmd
}
