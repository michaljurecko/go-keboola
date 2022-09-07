package load

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/telemetry"
	repositoryManifest "github.com/keboola/keboola-as-code/internal/pkg/template/repository/manifest"
)

type dependencies interface {
	Tracer() trace.Tracer
	Logger() log.Logger
}

func Run(ctx context.Context, fs filesystem.Fs, d dependencies) (m *repositoryManifest.Manifest, err error) {
	ctx, span := d.Tracer().Start(ctx, "kac.lib.operation.template.local.repository.manifest.load")
	defer telemetry.EndSpan(span, &err)

	logger := d.Logger()

	m, err = repositoryManifest.Load(fs)
	if err != nil {
		return nil, err
	}

	logger.Debugf(`Repository manifest loaded.`)
	return m, nil
}
