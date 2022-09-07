package list

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/telemetry"
)

type dependencies interface {
	Tracer() trace.Tracer
	Logger() log.Logger
}

func Run(ctx context.Context, branch *model.BranchState, d dependencies) (err error) {
	ctx, span := d.Tracer().Start(ctx, "kac.lib.operation.project.local.template.list")
	defer telemetry.EndSpan(span, &err)

	w := d.Logger().InfoWriter()

	// Get instances
	instances, err := branch.Local.Metadata.TemplatesInstances()
	if err != nil {
		return err
	}

	for _, instance := range instances {
		w.Writef("Template ID:          %s", instance.TemplateId)
		w.Writef("Instance ID:          %s", instance.InstanceId)
		w.Writef("RepositoryName:       %s", instance.RepositoryName)
		w.Writef("Version:              %s", instance.Version)
		w.Writef("Name:                 %s", instance.InstanceName)
		w.Writef("Created:")
		w.Writef("  Date:               %s", instance.Created.Date.Format(time.RFC3339))
		w.Writef("  TokenID:            %s", instance.Created.TokenId)
		w.Writef("Updated:")
		w.Writef("  Date:               %s", instance.Updated.Date.Format(time.RFC3339))
		w.Writef("  TokenID:            %s", instance.Updated.TokenId)
		w.Writef("")
	}

	return nil
}
