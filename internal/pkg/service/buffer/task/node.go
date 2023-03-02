package task

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	etcd "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"

	"github.com/keboola/keboola-as-code/internal/pkg/idgenerator"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/store/key"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/store/model"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/store/schema"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/etcdop"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/etcdop/op"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/servicectx"
	"github.com/keboola/keboola-as-code/internal/pkg/telemetry"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

const LockEtcdPrefix = etcdop.Prefix("runtime/lock/task")

// Node represents a cluster Worker node on which tasks are run.
// See comments in the StartTask method.
type Node struct {
	tracer   trace.Tracer
	clock    clock.Clock
	logger   log.Logger
	schema   *schema.Schema
	client   *etcd.Client
	tasksCtx context.Context
	tasksWg  *sync.WaitGroup

	sessionLock *sync.RWMutex
	session     *concurrency.Session

	nodeID           string
	config           config
	tasksCount       *atomic.Int64
	runningTasksLock *sync.Mutex
	runningTasks     map[key.TaskKey]bool
}

type Result = string

type Task func(ctx context.Context, logger log.Logger) (Result, error)

type dependencies interface {
	Tracer() trace.Tracer
	Clock() clock.Clock
	Logger() log.Logger
	Process() *servicectx.Process
	Schema() *schema.Schema
	EtcdClient() *etcd.Client
}

func NewNode(d dependencies, opts ...Option) (*Node, error) {
	// Apply options
	c := defaultConfig()
	for _, o := range opts {
		o(&c)
	}

	proc := d.Process()

	n := &Node{
		tracer:           d.Tracer(),
		clock:            d.Clock(),
		logger:           d.Logger().AddPrefix("[task]"),
		schema:           d.Schema(),
		client:           d.EtcdClient(),
		nodeID:           proc.UniqueID(),
		config:           c,
		tasksCount:       atomic.NewInt64(0),
		runningTasksLock: &sync.Mutex{},
		runningTasks:     make(map[key.TaskKey]bool),
	}

	// Graceful shutdown
	var cancelTasks context.CancelFunc
	n.tasksWg = &sync.WaitGroup{}
	n.tasksCtx, cancelTasks = context.WithCancel(context.Background())
	sessionWg := &sync.WaitGroup{}
	sessionCtx, cancelSession := context.WithCancel(context.Background())
	proc.OnShutdown(func() {
		n.logger.Info("received shutdown request")
		if c := n.tasksCount.Load(); c > 0 {
			n.logger.Infof(`waiting for "%d" tasks to be finished`, c)
		}
		cancelTasks()
		n.tasksWg.Wait()
		cancelSession()
		sessionWg.Wait()
		n.logger.Info("shutdown done")
	})

	// Create etcd session
	n.sessionLock = &sync.RWMutex{}
	sessionInit := etcdop.ResistantSession(sessionCtx, sessionWg, n.logger, n.client, c.ttlSeconds, func(session *concurrency.Session) error {
		n.sessionLock.Lock()
		n.session = session
		n.sessionLock.Unlock()
		return nil
	})

	if err := <-sessionInit; err != nil {
		return nil, err
	}

	return n, nil
}

func (n *Node) TasksCount() int64 {
	return n.tasksCount.Load()
}

// StartTask backed by local lock and etcd transaction, so the task run at most once.
// The context will be passed to the operation callback.
func (n *Node) StartTask(ctx context.Context, receiverKey key.ReceiverKey, typ, lock string, operation Task) (t *model.Task, err error) {
	createdAt := key.UTCTime(n.clock.Now())
	taskID := key.TaskID(fmt.Sprintf("%s_%s", createdAt.String(), idgenerator.Random(5)))
	taskKey := key.TaskKey{ReceiverKey: receiverKey, Type: typ, TaskID: taskID}

	// Lock task locally for periodical re-syncs,
	// so locally can be determined that the task is already running.
	ok, unlock := n.lockTask(taskKey)
	if !ok {
		return nil, nil
	}

	// Create task model
	task := model.Task{TaskKey: taskKey, CreatedAt: createdAt, WorkerNode: n.nodeID, Lock: lock}

	// Get session
	n.sessionLock.RLock()
	session := n.session
	n.sessionLock.RUnlock()

	// Create task and lock in etcd
	// Atomicity: If the lock key already exists, the then the transaction fails and task is ignored.
	// Resistance to outages: If the Worker node fails, the lock is released automatically by the lease, after the session TTL seconds.
	taskEtcdKey := n.schema.Tasks().ByKey(task.TaskKey)
	lockEtcdKey := n.schema.Runtime().Lock().Task().LockKey(task.Lock)
	logger := n.logger.AddPrefix(fmt.Sprintf("[%s]", taskKey.ID()))
	createTaskOp := op.MergeToTxn(
		taskEtcdKey.Put(task),
		lockEtcdKey.PutIfNotExists(task.WorkerNode, etcd.WithLease(session.Lease())),
	)
	if resp, err := createTaskOp.Do(n.tasksCtx, n.client); err != nil {
		unlock()
		return nil, errors.Errorf(`cannot start task "%s": %s`, taskKey, err)
	} else if !resp.Succeeded {
		unlock()
		logger.Infof(`task ignored, the lock "%s" is in use`, lockEtcdKey.Key())
		return nil, nil
	}

	logger.Infof(`started task "%s"`, taskKey)
	logger.Debugf(`lock acquired "%s"`, lockEtcdKey.Key())

	_, span := n.tracer.Start(ctx, "keboola.go.buffer.task."+task.Type)
	defer telemetry.EndSpan(span, &err)
	span.SetAttributes(
		telemetry.KeepSpan(),
		attribute.String("projectId", task.ProjectID.String()),
		attribute.String("receiverId", task.ReceiverID.String()),
		attribute.String("lock", task.Lock),
		attribute.String("worker", task.WorkerNode),
		attribute.String("createdAt", task.CreatedAt.String()),
	)
	createdTask := task

	// Run operation in the background
	go func() {
		defer unlock()

		// Process results, in defer, to catch panic
		var result string
		var err error
		startTime := n.clock.Now()
		defer func() {
			// Catch panic
			if panicErr := recover(); panicErr != nil {
				result = ""
				err = errors.Errorf("panic: %s, stacktrace: %s", panicErr, string(debug.Stack()))
				logger.Errorf(`task panic: %s`, err)
			}

			// Calculate duration
			endTime := n.clock.Now()
			finishedAt := key.UTCTime(endTime)
			duration := endTime.Sub(startTime)

			// Update fields
			task.FinishedAt = &finishedAt
			task.Duration = &duration
			if err == nil {
				task.Result = result
				logger.Infof(`task succeeded (%s): %s`, duration, result)
			} else {
				task.Error = err.Error()
				logger.Warnf(`task failed (%s): %s`, duration, errors.Format(err, errors.FormatWithStack()))
			}
			span.SetAttributes(
				attribute.Float64("duration", task.Duration.Seconds()),
				attribute.String("result", task.Result),
				attribute.String("finishedAt", task.FinishedAt.String()),
			)

			// If release of the lock takes longer than the ttl, lease is expired anyway
			opCtx, cancel := context.WithTimeout(context.Background(), time.Duration(n.config.ttlSeconds)*time.Second)
			defer cancel()

			// Update task and release lock in etcd
			finishTaskOp := op.MergeToTxn(
				taskEtcdKey.Put(task),
				lockEtcdKey.DeleteIfExists(),
			)
			if resp, err := finishTaskOp.Do(opCtx, n.client); err != nil {
				logger.Errorf(`cannot update task and release lock: %s`, err)
				return
			} else if !resp.Succeeded {
				logger.Errorf(`cannot release task lock "%s", not found`, lockEtcdKey.Key())
				return
			}
			logger.Debugf(`lock released "%s"`, lockEtcdKey.Key())
		}()

		// Do operation
		result, err = operation(ctx, logger)
	}()
	return &createdTask, nil
}

// lockTask guarantees that the task runs at most once on the Worker node.
// Uniqueness within the cluster is guaranteed by the etcd transaction, see StartTask method.
func (n *Node) lockTask(taskKey key.TaskKey) (ok bool, unlock func()) {
	n.runningTasksLock.Lock()
	defer n.runningTasksLock.Unlock()
	if n.runningTasks[taskKey] {
		return false, nil
	}

	n.tasksWg.Add(1)
	n.tasksCount.Inc()
	n.runningTasks[taskKey] = true

	return true, func() {
		n.runningTasksLock.Lock()
		defer n.runningTasksLock.Unlock()
		delete(n.runningTasks, taskKey)
		n.tasksCount.Dec()
		n.tasksWg.Done()
	}
}
