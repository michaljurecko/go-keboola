// Package dependencies provides dependencies for Buffer Service.
//
// # Dependency Containers
//
// This package extends common dependencies from [pkg/github.com/keboola/keboola-as-code/internal/pkg/service/common/dependencies].
//
// Following dependencies containers are implemented:
//   - [ServiceScope] contains common part of dependencies for all scopes.
//   - [APIScope] contains long-lived dependencies that exist during the entire run of an API node.
//   - [PublicRequestScope] contains short-lived dependencies for a public request without authentication.
//   - [ProjectRequestScope] contains short-lived dependencies for a request with authentication.
//   - [SourceScope] contains long-lived dependencies for source nodes.
//   - [StorageScope] contains long-lived dependencies for local storage writer/reader nodes.
//
// Dependency containers creation:
//   - [ServiceScope] is created during the creation of [APIScope] or [StorageScope].
//   - [APIScope] is created at startup in the API main.go.
//   - [PublicRequestScope] is created for each HTTP request by Muxer.Use callback in main.go.
//   - [ProjectRequestScope] is created for each authenticated HTTP request in the service.APIKeyAuth method.
//   - [SourceScope] is created at startup of a source node.
//   - [CoordinatorScope] is created at startup of a coordinator node.
//   - [StorageScope] is created at startup of a local storage writer/reader node.
//   - [StorageWriterScope] is created at startup of a local storage writer node.
//   - [StorageReaderScope] is created at startup of a local storage reader node.
//
// The package also provides mocked dependency implementations for tests:
//   - [NewMockedServiceScope]
//   - [NewMockedAPIScope]
//   - [NewMockedPublicRequestScope]
//   - [NewMockedProjectRequestScope]
//   - [NewMockedCoordinatorScope]
//   - [NewMockedStorageScope]
//   - [NewMockedStorageWriterScope]
//   - [NewMockedStorageReaderScope]
//
// Dependencies injection to service endpoints:
//   - Each service endpoint method gets [PublicRequestScope] as a parameter.
//   - Authorized endpoints gets [ProjectRequestScope] instead.
//   - The injection is generated by "internal/pkg/service/common/goaextension/dependencies" package.
//   - See service implementation for details [src/github.com/keboola/keboola-as-code/internal/pkg/service/biffer/api/service/service.go].
package dependencies

import (
	"net/url"

	"github.com/keboola/keboola-as-code/internal/pkg/service/common/dependencies"
	aggregationRepo "github.com/keboola/keboola-as-code/internal/pkg/service/stream/aggregation/repository"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/config"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/definition"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/definition/key"
	definitionRepo "github.com/keboola/keboola-as-code/internal/pkg/service/stream/definition/repository"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/plugin"
	sinkRouter "github.com/keboola/keboola-as-code/internal/pkg/service/stream/sink/router"
	keboolaSinkBridge "github.com/keboola/keboola-as-code/internal/pkg/service/stream/sink/type/tablesink/keboola/bridge"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/level/local/diskreader"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/level/local/diskwriter"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/level/local/diskwriter/network/connection"
	storageRouter "github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/level/local/diskwriter/network/router"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/level/local/encoding"
	storageRepo "github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/model/repository"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/statistics/cache"
	statsRepo "github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/statistics/repository"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/storage/test/dummy"
)

type ctxKey string

const (
	PublicRequestScopeCtxKey  = ctxKey("PublicRequestScope")
	ProjectRequestScopeCtxKey = ctxKey("ProjectRequestScope")
	BranchRequestScopeCtxKey  = ctxKey("BranchRequestScope")
	SourceRequestScopeCtxKey  = ctxKey("SourceRequestScope")
	SinkRequestScopeCtxKey    = ctxKey("SinkRequestScope")
	KeboolaProjectAPICtxKey   = ctxKey("KeboolaAuthorizedAPI")
)

type ServiceScope interface {
	dependencies.BaseScope
	dependencies.PublicScope
	dependencies.EtcdClientScope
	Plugins() *plugin.Plugins
	DefinitionRepository() *definitionRepo.Repository
	StorageRepository() *storageRepo.Repository
	StatisticsRepository() *statsRepo.Repository
	AggregationRepository() *aggregationRepo.Repository
	KeboolaSinkBridge() *keboolaSinkBridge.Bridge
}

type APIScope interface {
	ServiceScope
	dependencies.TaskScope
	dependencies.DistributedLockScope
	APIPublicURL() *url.URL
	HTTPSourcePublicURL() *url.URL
}

type PublicRequestScope interface {
	APIScope
	dependencies.RequestInfo
}

type ProjectRequestScope interface {
	PublicRequestScope
	dependencies.ProjectScope
	RequestUser() definition.By
}

type BranchRequestScope interface {
	ProjectRequestScope
	Branch() definition.Branch
	BranchKey() key.BranchKey
}

type SourceRequestScope interface {
	BranchRequestScope
	SourceKey() key.SourceKey
}

type SinkRequestScope interface {
	SourceRequestScope
	SinkKey() key.SinkKey
}

type SourceScope interface {
	ServiceScope
	dependencies.DistributionScope
	EncodingManager() *encoding.Manager
	ConnectionManager() *connection.Manager
	SinkRouter() *sinkRouter.Router
	StorageRouter() *storageRouter.Router
}

type CoordinatorScope interface {
	ServiceScope
	dependencies.DistributionScope
	dependencies.DistributedLockScope
	StatisticsL1Cache() *cache.L1
	StatisticsL2Cache() *cache.L2
}

type StorageScope interface {
	ServiceScope
}

type StorageWriterScope interface {
	StorageScope
	dependencies.DistributionScope
	Volumes() *diskwriter.Volumes
}

type StorageReaderScope interface {
	StorageScope
	Volumes() *diskreader.Volumes
}

type Mocked interface {
	dependencies.Mocked
	TestConfig() config.Config
	TestDummySinkController() *dummy.SinkController
}
