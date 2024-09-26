package task_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/keboola/keboola-as-code/internal/pkg/service/common/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/distribution"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/etcdop"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/task"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/utctime"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/etcdhelper"
)

type testDependencies struct {
	dependencies.Mocked
	dependencies.DistributionScope
}

func TestCleanup(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clk := clock.NewMock()
	mock := dependencies.NewMocked(t, ctx, dependencies.WithClock(clk), dependencies.WithEnabledEtcdClient())
	client := mock.TestEtcdClient()
	d := testDependencies{
		Mocked:            mock,
		DistributionScope: dependencies.NewDistributionScope("my-node", distribution.NewConfig(), mock),
	}

	taskPrefix := etcdop.NewTypedPrefix[task.Task](task.TaskEtcdPrefix, d.EtcdSerde())

	cleanupInterval := 15 * time.Second
	require.NoError(t, task.StartCleaner(d, cleanupInterval))

	// Add task without a finishedAt timestamp but too old - will be deleted
	createdAtRaw, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05+07:00")
	createdAt := utctime.UTCTime(createdAtRaw)
	taskKey1 := task.Key{ProjectID: 123, TaskID: task.ID(fmt.Sprintf("%s/%s_%s", "some.task", createdAt.String(), "abcdef"))}
	task1 := task.Task{
		Key:        taskKey1,
		Type:       "some.task",
		CreatedAt:  createdAt,
		FinishedAt: nil,
		Node:       "node1",
		Lock:       "lock1",
		Result:     "",
		Error:      "err",
		Duration:   nil,
	}
	assert.NoError(t, taskPrefix.Key(taskKey1.String()).Put(client, task1).Do(ctx).Err())

	// Add task with a finishedAt timestamp in the past - will be deleted
	time2, _ := time.Parse(time.RFC3339, "2008-01-02T15:04:05+07:00")
	time2Key := utctime.UTCTime(time2)
	taskKey2 := task.Key{ProjectID: 456, TaskID: task.ID(fmt.Sprintf("%s/%s_%s", "other.task", createdAt.String(), "ghijkl"))}
	task2 := task.Task{
		Key:        taskKey2,
		Type:       "other.task",
		CreatedAt:  createdAt,
		FinishedAt: &time2Key,
		Node:       "node2",
		Lock:       "lock2",
		Result:     "res",
		Error:      "",
		Duration:   nil,
	}
	assert.NoError(t, taskPrefix.Key(taskKey2.String()).Put(client, task2).Do(ctx).Err())

	// Add task with a finishedAt timestamp before a moment - will be ignored
	time3 := time.Now()
	time3Key := utctime.UTCTime(time3)
	taskKey3 := task.Key{ProjectID: 789, TaskID: task.ID(fmt.Sprintf("%s/%s_%s", "third.task", createdAt.String(), "mnopqr"))}
	task3 := task.Task{
		Key:        taskKey3,
		Type:       "third.task",
		CreatedAt:  createdAt,
		FinishedAt: &time3Key,
		Node:       "node2",
		Lock:       "lock2",
		Result:     "res",
		Error:      "",
		Duration:   nil,
	}
	assert.NoError(t, taskPrefix.Key(taskKey3.String()).Put(client, task3).Do(ctx).Err())

	// Run the cleanup
	clk.Add(cleanupInterval)

	// Shutdown - wait for cleanup
	d.Process().Shutdown(ctx, errors.New("bye bye"))
	d.Process().WaitForShutdown()

	// Check logs
	d.DebugLogger().AssertJSONMessages(t, `
{"level":"info","message":"started task","component":"task","task":"_system_/tasks.cleanup/%s","node":"node1"}
{"level":"debug","message":"lock acquired \"runtime/lock/task/tasks.cleanup\"","component":"task","task":"_system_/tasks.cleanup/%s","node":"node1"}
{"level":"debug","message":"deleted task \"123/some.task/2006-01-02T08:04:05.000Z_abcdef\"","component":"task","task":"_system_/tasks.cleanup/%s","node":"node1"}
{"level":"debug","message":"deleted task \"456/other.task/2006-01-02T08:04:05.000Z_ghijkl\"","component":"task","task":"_system_/tasks.cleanup/%s","node":"node1"}
{"level":"info","message":"deleted \"2\" tasks","component":"task","task":"_system_/tasks.cleanup/%s","node":"node1"}
{"level":"info","message":"task succeeded (%s): deleted \"2\" tasks","component":"task","task":"_system_/tasks.cleanup/%s","node":"node1"}
{"level":"debug","message":"lock released \"runtime/lock/task/tasks.cleanup\"","component":"task","task":"_system_/tasks.cleanup/%s","node":"node1"}
{"level":"info","message":"exiting (bye bye)"}
{"level":"info","message":"received shutdown request","component":"task","node":"node1"}
{"level":"info","message":"closing etcd session: context canceled","component":"task.etcd.session","node":"node1"}
{"level":"info","message":"closed etcd session","component":"task.etcd.session","node":"node1"}
{"level":"info","message":"shutdown done","component":"task","node":"node1"}
{"level":"info","message":"closing etcd connection","component":"etcd.client"}
{"level":"info","message":"closed etcd connection","component":"etcd.client"}
{"level":"info","message":"exited"}
`)

	// Check keys
	etcdhelper.AssertKVsString(t, client, `
<<<<<
task/789/third.task/2006-01-02T08:04:05.000Z_mnopqr
-----
{
  "projectId": 789,
  "taskId": "third.task/2006-01-02T08:04:05.000Z_mnopqr",
  "type": "third.task",
  "createdAt": "2006-01-02T08:04:05.000Z",
  "finishedAt": "%s",
  "node": "node2",
  "lock": "lock2",
  "result": "res"
}
>>>>>

<<<<<
task/_system_/tasks.cleanup/%s
-----
{
  "systemTask": true,
  "taskId": "tasks.cleanup/%s",
  "type": "tasks.cleanup",
  "createdAt": "%s",
  "finishedAt": "%s",
  "node": "node1",
  "lock": "runtime/lock/task/tasks.cleanup",
  "result": "deleted \"2\" tasks",
  "duration": %d
}
>>>>>
`)
}
