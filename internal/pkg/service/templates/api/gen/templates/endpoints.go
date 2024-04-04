// Code generated by goa v3.15.2, DO NOT EDIT.
//
// templates endpoints
//
// Command:
// $ goa gen github.com/keboola/keboola-as-code/api/templates --output
// ./internal/pkg/service/templates/api

package templates

import (
	"context"

	dependencies "github.com/keboola/keboola-as-code/internal/pkg/service/templates/dependencies"
	goa "goa.design/goa/v3/pkg"
	"goa.design/goa/v3/security"
)

// Endpoints wraps the "templates" service endpoints.
type Endpoints struct {
	APIRootIndex                  goa.Endpoint
	APIVersionIndex               goa.Endpoint
	HealthCheck                   goa.Endpoint
	RepositoriesIndex             goa.Endpoint
	RepositoryIndex               goa.Endpoint
	TemplatesIndex                goa.Endpoint
	TemplateIndex                 goa.Endpoint
	VersionIndex                  goa.Endpoint
	InputsIndex                   goa.Endpoint
	ValidateInputs                goa.Endpoint
	UseTemplateVersion            goa.Endpoint
	InstancesIndex                goa.Endpoint
	InstanceIndex                 goa.Endpoint
	UpdateInstance                goa.Endpoint
	DeleteInstance                goa.Endpoint
	UpgradeInstance               goa.Endpoint
	UpgradeInstanceInputsIndex    goa.Endpoint
	UpgradeInstanceValidateInputs goa.Endpoint
	GetTask                       goa.Endpoint
}

// NewEndpoints wraps the methods of the "templates" service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	// Casting service to Auther interface
	a := s.(Auther)
	return &Endpoints{
		APIRootIndex:                  NewAPIRootIndexEndpoint(s),
		APIVersionIndex:               NewAPIVersionIndexEndpoint(s),
		HealthCheck:                   NewHealthCheckEndpoint(s),
		RepositoriesIndex:             NewRepositoriesIndexEndpoint(s, a.APIKeyAuth),
		RepositoryIndex:               NewRepositoryIndexEndpoint(s, a.APIKeyAuth),
		TemplatesIndex:                NewTemplatesIndexEndpoint(s, a.APIKeyAuth),
		TemplateIndex:                 NewTemplateIndexEndpoint(s, a.APIKeyAuth),
		VersionIndex:                  NewVersionIndexEndpoint(s, a.APIKeyAuth),
		InputsIndex:                   NewInputsIndexEndpoint(s, a.APIKeyAuth),
		ValidateInputs:                NewValidateInputsEndpoint(s, a.APIKeyAuth),
		UseTemplateVersion:            NewUseTemplateVersionEndpoint(s, a.APIKeyAuth),
		InstancesIndex:                NewInstancesIndexEndpoint(s, a.APIKeyAuth),
		InstanceIndex:                 NewInstanceIndexEndpoint(s, a.APIKeyAuth),
		UpdateInstance:                NewUpdateInstanceEndpoint(s, a.APIKeyAuth),
		DeleteInstance:                NewDeleteInstanceEndpoint(s, a.APIKeyAuth),
		UpgradeInstance:               NewUpgradeInstanceEndpoint(s, a.APIKeyAuth),
		UpgradeInstanceInputsIndex:    NewUpgradeInstanceInputsIndexEndpoint(s, a.APIKeyAuth),
		UpgradeInstanceValidateInputs: NewUpgradeInstanceValidateInputsEndpoint(s, a.APIKeyAuth),
		GetTask:                       NewGetTaskEndpoint(s, a.APIKeyAuth),
	}
}

// Use applies the given middleware to all the "templates" service endpoints.
func (e *Endpoints) Use(m func(goa.Endpoint) goa.Endpoint) {
	e.APIRootIndex = m(e.APIRootIndex)
	e.APIVersionIndex = m(e.APIVersionIndex)
	e.HealthCheck = m(e.HealthCheck)
	e.RepositoriesIndex = m(e.RepositoriesIndex)
	e.RepositoryIndex = m(e.RepositoryIndex)
	e.TemplatesIndex = m(e.TemplatesIndex)
	e.TemplateIndex = m(e.TemplateIndex)
	e.VersionIndex = m(e.VersionIndex)
	e.InputsIndex = m(e.InputsIndex)
	e.ValidateInputs = m(e.ValidateInputs)
	e.UseTemplateVersion = m(e.UseTemplateVersion)
	e.InstancesIndex = m(e.InstancesIndex)
	e.InstanceIndex = m(e.InstanceIndex)
	e.UpdateInstance = m(e.UpdateInstance)
	e.DeleteInstance = m(e.DeleteInstance)
	e.UpgradeInstance = m(e.UpgradeInstance)
	e.UpgradeInstanceInputsIndex = m(e.UpgradeInstanceInputsIndex)
	e.UpgradeInstanceValidateInputs = m(e.UpgradeInstanceValidateInputs)
	e.GetTask = m(e.GetTask)
}

// NewAPIRootIndexEndpoint returns an endpoint function that calls the method
// "ApiRootIndex" of service "templates".
func NewAPIRootIndexEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		deps := ctx.Value(dependencies.PublicRequestScopeCtxKey).(dependencies.PublicRequestScope)
		return nil, s.APIRootIndex(ctx, deps)
	}
}

// NewAPIVersionIndexEndpoint returns an endpoint function that calls the
// method "ApiVersionIndex" of service "templates".
func NewAPIVersionIndexEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		deps := ctx.Value(dependencies.PublicRequestScopeCtxKey).(dependencies.PublicRequestScope)
		return s.APIVersionIndex(ctx, deps)
	}
}

// NewHealthCheckEndpoint returns an endpoint function that calls the method
// "HealthCheck" of service "templates".
func NewHealthCheckEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		deps := ctx.Value(dependencies.PublicRequestScopeCtxKey).(dependencies.PublicRequestScope)
		return s.HealthCheck(ctx, deps)
	}
}

// NewRepositoriesIndexEndpoint returns an endpoint function that calls the
// method "RepositoriesIndex" of service "templates".
func NewRepositoriesIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*RepositoriesIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.RepositoriesIndex(ctx, deps, p)
	}
}

// NewRepositoryIndexEndpoint returns an endpoint function that calls the
// method "RepositoryIndex" of service "templates".
func NewRepositoryIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*RepositoryIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.RepositoryIndex(ctx, deps, p)
	}
}

// NewTemplatesIndexEndpoint returns an endpoint function that calls the method
// "TemplatesIndex" of service "templates".
func NewTemplatesIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*TemplatesIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.TemplatesIndex(ctx, deps, p)
	}
}

// NewTemplateIndexEndpoint returns an endpoint function that calls the method
// "TemplateIndex" of service "templates".
func NewTemplateIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*TemplateIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.TemplateIndex(ctx, deps, p)
	}
}

// NewVersionIndexEndpoint returns an endpoint function that calls the method
// "VersionIndex" of service "templates".
func NewVersionIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*VersionIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.VersionIndex(ctx, deps, p)
	}
}

// NewInputsIndexEndpoint returns an endpoint function that calls the method
// "InputsIndex" of service "templates".
func NewInputsIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*InputsIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.InputsIndex(ctx, deps, p)
	}
}

// NewValidateInputsEndpoint returns an endpoint function that calls the method
// "ValidateInputs" of service "templates".
func NewValidateInputsEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*ValidateInputsPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.ValidateInputs(ctx, deps, p)
	}
}

// NewUseTemplateVersionEndpoint returns an endpoint function that calls the
// method "UseTemplateVersion" of service "templates".
func NewUseTemplateVersionEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*UseTemplateVersionPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.UseTemplateVersion(ctx, deps, p)
	}
}

// NewInstancesIndexEndpoint returns an endpoint function that calls the method
// "InstancesIndex" of service "templates".
func NewInstancesIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*InstancesIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.InstancesIndex(ctx, deps, p)
	}
}

// NewInstanceIndexEndpoint returns an endpoint function that calls the method
// "InstanceIndex" of service "templates".
func NewInstanceIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*InstanceIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.InstanceIndex(ctx, deps, p)
	}
}

// NewUpdateInstanceEndpoint returns an endpoint function that calls the method
// "UpdateInstance" of service "templates".
func NewUpdateInstanceEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*UpdateInstancePayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.UpdateInstance(ctx, deps, p)
	}
}

// NewDeleteInstanceEndpoint returns an endpoint function that calls the method
// "DeleteInstance" of service "templates".
func NewDeleteInstanceEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*DeleteInstancePayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.DeleteInstance(ctx, deps, p)
	}
}

// NewUpgradeInstanceEndpoint returns an endpoint function that calls the
// method "UpgradeInstance" of service "templates".
func NewUpgradeInstanceEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*UpgradeInstancePayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.UpgradeInstance(ctx, deps, p)
	}
}

// NewUpgradeInstanceInputsIndexEndpoint returns an endpoint function that
// calls the method "UpgradeInstanceInputsIndex" of service "templates".
func NewUpgradeInstanceInputsIndexEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*UpgradeInstanceInputsIndexPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.UpgradeInstanceInputsIndex(ctx, deps, p)
	}
}

// NewUpgradeInstanceValidateInputsEndpoint returns an endpoint function that
// calls the method "UpgradeInstanceValidateInputs" of service "templates".
func NewUpgradeInstanceValidateInputsEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*UpgradeInstanceValidateInputsPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.UpgradeInstanceValidateInputs(ctx, deps, p)
	}
}

// NewGetTaskEndpoint returns an endpoint function that calls the method
// "GetTask" of service "templates".
func NewGetTaskEndpoint(s Service, authAPIKeyFn security.AuthAPIKeyFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*GetTaskPayload)
		var err error
		sc := security.APIKeyScheme{
			Name:           "storage-api-token",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authAPIKeyFn(ctx, p.StorageAPIToken, &sc)
		if err != nil {
			return nil, err
		}
		deps := ctx.Value(dependencies.ProjectRequestScopeCtxKey).(dependencies.ProjectRequestScope)
		return s.GetTask(ctx, deps, p)
	}
}
