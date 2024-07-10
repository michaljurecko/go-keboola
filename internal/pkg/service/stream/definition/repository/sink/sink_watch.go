package sink

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/keboola/keboola-as-code/internal/pkg/service/common/etcdop"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/definition"
)

func (r *Repository) GetAllAndWatch(ctx context.Context, opts ...clientv3.OpOption) *etcdop.RestartableWatchStreamT[definition.Sink] {
	return r.schema.Active().GetAllAndWatch(ctx, r.client, opts...)
}
