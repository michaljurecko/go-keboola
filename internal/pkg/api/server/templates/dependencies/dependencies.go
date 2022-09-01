// Package dependencies provides dependencies for Templates API.
//
// # Dependency Containers
//
// This package extends common dependencies from [pkg/github.com/keboola/keboola-as-code/internal/pkg/dependencies].
//
// These dependencies containers are implemented:
//   - [ForServer] long-lived dependencies that exist during the entire run of the API server.
//   - [ForPublicRequest] short-lived dependencies for a public request that does not contain the Storage API token.
//   - [ForProjectRequest] short-lived dependencies for an authenticated request that contains the Storage API token.
package dependencies

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/keboola/go-client/pkg/client"
	etcd "go.etcd.io/etcd/client/v3"
	ddHttp "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/keboola/keboola-as-code/internal/pkg/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/env"
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/filesystem/aferofs"
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/template/repository"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/strhelper"
	loadRepositoryManifest "github.com/keboola/keboola-as-code/pkg/lib/operation/template/local/repository/manifest/load"
)

type ctxKey string

const ForPublicRequestCtxKey = ctxKey("ForPublicRequest")
const ForProjectRequestCtxKey = ctxKey("ForProjectRequest")
const EtcdTestConnectionTimeout = 1 * time.Second

// ForServer interface provides dependencies for Templates API server.
// The container exists during the entire run of the API server.
type ForServer interface {
	dependencies.Base
	dependencies.Public
	ServerCtx() context.Context
	PrefixLogger() log.PrefixLogger
	RepositoryManager() *repository.Manager
	EtcdClient() (*etcd.Client, error)
}

// ForPublicRequest interface provides dependencies for a public request that does not contain the Storage API token.
// The container exists only during request processing.
type ForPublicRequest interface {
	ForServer
	RequestCtx() context.Context
	RequestID() string
}

// ForProjectRequest interface provides dependencies for an authenticated request that contains the Storage API token.
// The container exists only during request processing.
type ForProjectRequest interface {
	ForPublicRequest
	dependencies.Project
	ProjectRepositories() []model.TemplateRepository
}

// forServer implements ForServer interface.
type forServer struct {
	dependencies.Base
	dependencies.Public
	serverCtx         context.Context
	logger            log.PrefixLogger
	repositoryManager *repository.Manager
	etcdClient        dependencies.Lazy[*etcd.Client]
}

// forPublicRequest implements ForPublicRequest interface.
type forPublicRequest struct {
	ForServer
	logger     log.PrefixLogger
	requestCtx context.Context
	requestID  string
	components dependencies.Lazy[*model.ComponentsMap]
}

// forProjectRequest implements ForProjectRequest interface.
type forProjectRequest struct {
	dependencies.Project
	ForPublicRequest
	logger              log.PrefixLogger
	projectRepositories dependencies.Lazy[[]model.TemplateRepository]
}

func NewServerDeps(serverCtx context.Context, envs env.Provider, logger log.PrefixLogger, defaultRepositories []model.TemplateRepository, debug, dumpHttp bool) (ForServer, error) {
	// Get Storage API host
	storageApiHost := strhelper.NormalizeHost(envs.MustGet("KBC_STORAGE_API_HOST"))
	if storageApiHost == "" {
		return nil, fmt.Errorf("KBC_STORAGE_API_HOST environment variable is not set")
	}

	// Create base HTTP client for all API requests to other APIs
	httpClient := apiHttpClient(envs, logger, debug, dumpHttp)

	// Create server dependencies
	baseDeps := dependencies.NewBaseDeps(envs, logger, httpClient)
	publicDeps, err := dependencies.NewPublicDeps(serverCtx, baseDeps, storageApiHost)
	if err != nil {
		return nil, err
	}

	// Create repository manager
	repositoryManager, err := repository.NewManager(serverCtx, logger, defaultRepositories)
	if err != nil {
		return nil, err
	}

	// Create server dependencies
	d := &forServer{
		Base:              baseDeps,
		Public:            publicDeps,
		serverCtx:         serverCtx,
		logger:            logger,
		repositoryManager: repositoryManager,
	}

	// Test connection to etcd at server startup
	if _, err := d.EtcdClient(); err != nil {
		d.Logger().Warnf("cannot connect to etcd: %s", err.Error())
	}

	return d, nil
}
func NewDepsForPublicRequest(serverDeps ForServer, requestCtx context.Context, requestId string) ForPublicRequest {
	return &forPublicRequest{
		ForServer:  serverDeps,
		logger:     serverDeps.PrefixLogger().WithAdditionalPrefix(fmt.Sprintf("[requestId=%s]", requestId)),
		requestCtx: requestCtx,
		requestID:  requestId,
	}
}

func NewDepsForProjectRequest(publicDeps ForPublicRequest, ctx context.Context, tokenStr string) (ForProjectRequest, error) {
	projectDeps, err := dependencies.NewProjectDeps(ctx, publicDeps, publicDeps, tokenStr)
	if err != nil {
		return nil, err
	}

	logger := publicDeps.PrefixLogger().WithAdditionalPrefix(
		fmt.Sprintf("[project=%d][token=%s]", projectDeps.ProjectID(), projectDeps.StorageApiTokenID()),
	)

	return &forProjectRequest{
		logger:           logger,
		Project:          projectDeps,
		ForPublicRequest: publicDeps,
	}, nil
}

func (v *forServer) ServerCtx() context.Context {
	return v.serverCtx
}

func (v *forServer) PrefixLogger() log.PrefixLogger {
	return v.logger
}

func (v *forServer) RepositoryManager() *repository.Manager {
	return v.repositoryManager
}

func (v *forServer) EtcdClient() (*etcd.Client, error) {
	return v.etcdClient.InitAndGet(func() (*etcd.Client, error) {
		// Get endpoint
		endpoint := v.Envs().Get("ETCD_ENDPOINT")
		if endpoint == "" {
			return nil, fmt.Errorf("ETCD_HOST is not set")
		}

		// Create client
		c, err := etcd.New(etcd.Config{
			Context:              v.serverCtx,
			Endpoints:            []string{endpoint},
			DialTimeout:          2 * time.Second,
			DialKeepAliveTimeout: 2 * time.Second,
			DialKeepAliveTime:    10 * time.Second,
			Username:             v.Envs().Get("ETCD_USERNAME"), // optional
			Password:             v.Envs().Get("ETCD_PASSWORD"), // optional
		})
		if err != nil {
			return nil, err
		}

		// Sync endpoints list from cluster (also serves as a connection check)
		syncCtx, syncCancelFn := context.WithTimeout(v.ServerCtx(), EtcdTestConnectionTimeout)
		defer syncCancelFn()
		if err := c.Sync(syncCtx); err != nil {
			c.Close()
			return nil, err
		}

		// Close client when shutting down the server
		go func() {
			<-v.serverCtx.Done()
			if err := c.Close(); err != nil {
				v.Logger().Info("closed connection to etcd")
			} else {
				v.Logger().Warnf("cannot close connection etcd: %s", err)
			}
		}()

		v.logger.Infof(`connected to etcd cluster "%s"`, c.Endpoints()[0])
		return c, nil
	})
}

func (v *forPublicRequest) Logger() log.Logger {
	return v.logger
}

func (v *forPublicRequest) PrefixLogger() log.PrefixLogger {
	return v.logger
}

func (v *forPublicRequest) RequestCtx() context.Context {
	return v.requestCtx
}

func (v *forPublicRequest) RequestID() string {
	return v.requestID
}

func (v *forPublicRequest) Components() *model.ComponentsMap {
	// Use the same version of the components during the entire request
	return v.components.MustInitAndGet(func() *model.ComponentsMap {
		return v.ForServer.Components()
	})
}

func (v *forPublicRequest) TemplateRepository(_ context.Context, _ model.TemplateRepository, _ model.TemplateRef) (*repository.Repository, error) {
	panic(fmt.Errorf("template repositories depend on project features in the API, please use dependencies.ForProjectRequest instead of dependencies.ForPublicRequest"))
}

func (v *forProjectRequest) Logger() log.Logger {
	return v.logger
}

func (v *forProjectRequest) PrefixLogger() log.PrefixLogger {
	return v.logger
}

func (v *forProjectRequest) TemplateRepository(ctx context.Context, definition model.TemplateRepository, _ model.TemplateRef) (*repository.Repository, error) {
	var fs filesystem.Fs
	var err error
	if definition.Type == model.RepositoryTypeGit {
		// Get git repository
		gitRepository, err := v.RepositoryManager().Repository(definition)
		if err != nil {
			return nil, err
		}

		// Acquire read lock and release it after request,
		// so pull cannot occur in the middle of the request.
		gitRepository.RLock()
		go func() {
			<-v.RequestCtx().Done()
			gitRepository.RUnlock()
		}()
		fs = gitRepository.Fs()
	} else {
		fs, err = aferofs.NewLocalFs(v.Logger(), definition.Url, ".")
		if err != nil {
			return nil, err
		}
	}

	// Load manifest from FS
	manifest, err := loadRepositoryManifest.Run(ctx, fs, v)
	if err != nil {
		return nil, err
	}

	// Return repository
	return repository.New(definition, fs, manifest)
}

func (v *forProjectRequest) ProjectRepositories() []model.TemplateRepository {
	return v.projectRepositories.MustInitAndGet(func() []model.TemplateRepository {
		// Project repositories are default repositories modified by the project features.
		features := v.ProjectFeatures()
		var out []model.TemplateRepository
		for _, repo := range v.RepositoryManager().DefaultRepositories() {
			if repo.Name == repository.DefaultTemplateRepositoryName && repo.Ref == repository.DefaultTemplateRepositoryRefMain {
				if features.Has(repository.FeatureTemplateRepositoryBeta) {
					repo.Ref = repository.DefaultTemplateRepositoryRefBeta
				} else if features.Has(repository.FeatureTemplateRepositoryDev) {
					repo.Ref = repository.DefaultTemplateRepositoryRefDev
				}
			}
			out = append(out, repo)
		}

		return out
	})
}

func apiHttpClient(envs env.Provider, logger log.Logger, debug, dumpHttp bool) client.Client {
	// Force HTTP2 transport
	transport := client.HTTP2Transport()

	// DataDog low-level tracing
	if envs.Get("DATADOG_ENABLED") != "false" {
		transport = ddHttp.WrapRoundTripper(transport)
	}

	// Create client
	c := client.New().
		WithTransport(transport).
		WithUserAgent("keboola-templates-api")

	// Log each HTTP client request/response as debug message
	if debug {
		c = c.AndTrace(client.LogTracer(logger.DebugWriter()))
	}

	// Dump each HTTP client request/response body
	if dumpHttp {
		c = c.AndTrace(client.DumpTracer(logger.DebugWriter()))
	}

	// DataDog high-level tracing
	if envs.Get("DATADOG_ENABLED") != "false" {
		c = c.AndTrace(ddApiClientTrace())
	}

	return c
}

func ddApiClientTrace() client.TraceFactory {
	return func() *client.Trace {
		t := &client.Trace{}

		// Api request
		var ctx context.Context
		var apiReqSpan tracer.Span
		t.GotRequest = func(c context.Context, request client.HTTPRequest) context.Context {
			resultType := reflect.TypeOf(request.ResultDef())
			resultTypeString := ""
			if resultType != nil {
				resultTypeString = resultType.String()
			}
			apiReqSpan, ctx = tracer.StartSpanFromContext(
				c,
				"api.client.request",
				tracer.ResourceName("request"),
				tracer.SpanType("api.client"),
				tracer.Tag("result_type", resultTypeString),
			)
			return ctx
		}
		t.RequestProcessed = func(result any, err error) {
			apiReqSpan.Finish(tracer.WithError(err))
		}

		// Retry
		var retrySpan tracer.Span
		t.HTTPRequestRetry = func(attempt int, delay time.Duration) {
			retrySpan, _ = tracer.StartSpanFromContext(
				ctx,
				"api.client.retry.delay",
				tracer.ResourceName("retry"),
				tracer.SpanType("api.client"),
				tracer.Tag("attempt", attempt),
				tracer.Tag("delay_ms", delay.Milliseconds()),
				tracer.Tag("delay_string", delay.String()),
			)
		}
		t.HTTPRequestStart = func(r *http.Request) {
			if retrySpan != nil {
				apiReqSpan.Finish()
				retrySpan = nil
			}
		}
		return t
	}
}
