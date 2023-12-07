package status

import (
	"context"

	"github.com/keboola/keboola-as-code/internal/pkg/dbt"
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/project"
	"github.com/keboola/keboola-as-code/internal/pkg/telemetry"
	"github.com/keboola/keboola-as-code/internal/pkg/template"
	"github.com/keboola/keboola-as-code/internal/pkg/template/repository"
)

type dependencies interface {
	Fs() filesystem.Fs
	LocalProject(ignoreErrors bool) (*project.Project, bool, error)
	LocalTemplate(ctx context.Context) (*template.Template, bool, error)
	LocalTemplateRepository(ctx context.Context) (*repository.Repository, bool, error)
	LocalDbtProject(ctx context.Context) (*dbt.Project, bool, error)
	Logger() log.Logger
	Telemetry() telemetry.Telemetry
}

func Run(ctx context.Context, d dependencies) (err error) {
	ctx, span := d.Telemetry().Tracer().Start(ctx, "keboola.go.operation.status")
	defer span.End(&err)

	logger := d.Logger()

	if prj, found, err := d.LocalProject(false); found {
		if err != nil {
			return err
		}

		logger.InfofCtx(ctx, "Project directory:  %s", prj.Fs().BasePath())
		logger.InfofCtx(ctx, "Working directory:  %s", prj.Fs().WorkingDir())
		logger.InfofCtx(ctx, "Manifest path:      %s", prj.Manifest().Path())
		return nil
	}

	if tmpl, found, err := d.LocalTemplate(ctx); found {
		if err != nil {
			return err
		}

		logger.InfofCtx(ctx, "Template directory:  %s", tmpl.Fs().BasePath())
		logger.InfofCtx(ctx, "Working directory:   %s", tmpl.Fs().WorkingDir())
		logger.InfofCtx(ctx, "Manifest path:       %s", tmpl.ManifestPath())
		return nil
	}

	if repo, found, err := d.LocalTemplateRepository(ctx); found {
		if err != nil {
			return err
		}

		logger.InfofCtx(ctx, "Repository directory:  %s", repo.Fs().BasePath())
		logger.InfofCtx(ctx, "Working directory:     %s", repo.Fs().WorkingDir())
		logger.InfofCtx(ctx, "Manifest path:         %s", repo.Manifest().Path())
		return nil
	}

	if prj, found, err := d.LocalDbtProject(ctx); found {
		if err != nil {
			return err
		}

		logger.InfofCtx(ctx, "Dbt project directory:  %s", prj.Fs().BasePath())
		logger.InfofCtx(ctx, "Working directory:      %s", prj.Fs().WorkingDir())
		return nil
	}

	logger.WarnfCtx(ctx, `Directory "%s" is not a project or template repository.`, d.Fs().BasePath())
	return nil
}
