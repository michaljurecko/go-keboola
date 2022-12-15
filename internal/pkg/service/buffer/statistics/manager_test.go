package statistics

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"

	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/store/key"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/store/model"
)

func TestStatsManager(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clock := clock.NewMock()

	// mock store which contains every version of `SliceStats`
	store := newMockStatsStore()
	stats := New(ctx, func(_ context.Context, s []model.SliceStats) {
		store.append(s...)
	})

	// no notify -> wait 1 second -> no sync
	clock.Add(time.Second)
	assert.Empty(t, store.read())

	// notify -> wait 1 second -> sync
	receivedAt0 := clock.Now()
	k := key.NewSliceStatsKey(123, "my-receiver", "my-export", receivedAt0, receivedAt0, "my-node")
	stats.Notify(k, 1000)
	clock.Add(time.Second)
	assert.Equal(t,
		[]model.SliceStats{
			{
				SliceStatsKey:  k,
				Count:          1,
				Size:           1000,
				LastReceivedAt: receivedAt0,
			},
		},
		store.read(),
	)

	// no notify -> wait 1 second -> no sync
	clock.Add(time.Second)
	assert.Equal(t,
		[]model.SliceStats{
			{
				SliceStatsKey:  k,
				Count:          1,
				Size:           1000,
				LastReceivedAt: receivedAt0,
			},
		},
		store.read(),
	)

	// notify -> wait 1 second -> sync
	receivedAt1 := clock.Now()
	stats.Notify(k, 2000)
	clock.Add(time.Second)
	assert.Equal(t,
		[]model.SliceStats{
			{
				SliceStatsKey:  k,
				Count:          1,
				Size:           1000,
				LastReceivedAt: receivedAt0,
			},
			{
				SliceStatsKey:  k,
				Count:          2,
				Size:           3000,
				LastReceivedAt: receivedAt1,
			},
		},
		store.read(),
	)

	// no notify -> wait 1 second -> no sync
	clock.Add(time.Second)
	assert.Equal(t,
		[]model.SliceStats{
			{
				SliceStatsKey:  k,
				Count:          1,
				Size:           1000,
				LastReceivedAt: receivedAt0,
			},
			{
				SliceStatsKey:  k,
				Count:          2,
				Size:           3000,
				LastReceivedAt: receivedAt1,
			},
		},
		store.read(),
	)
}

type mockStatsStore struct {
	v []model.SliceStats
	m *sync.Mutex
}

func newMockStatsStore() mockStatsStore {
	return mockStatsStore{
		v: make([]model.SliceStats, 0),
		m: &sync.Mutex{},
	}
}

func (m *mockStatsStore) read() []model.SliceStats {
	m.m.Lock()
	defer m.m.Unlock()

	out := make([]model.SliceStats, len(m.v))
	copy(out, m.v)
	return out
}

func (m *mockStatsStore) append(v ...model.SliceStats) {
	m.m.Lock()
	defer m.m.Unlock()

	m.v = append(m.v, v...)
}
