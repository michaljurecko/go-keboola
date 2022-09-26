package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/git"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/template"
	"github.com/keboola/keboola-as-code/internal/pkg/template/repository"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
	pullOp "github.com/keboola/keboola-as-code/pkg/lib/operation/repository/pull"
	loadTemplateOp "github.com/keboola/keboola-as-code/pkg/lib/operation/template/load"
	loadRepositoryOp "github.com/keboola/keboola-as-code/pkg/lib/operation/template/repository/load"
)

// CachedRepository combines low-level git.Repository and high-level *repository.Repository.
// It adds caching support and coordinates template loading.
type CachedRepository struct {
	d dependencies

	git  git.Repository // *git.LocalRepository (type=dir) or *git.RemoteRepository (type=git)
	repo *repository.Repository

	templates     map[string]*template.Template
	templatesInit *singleflight.Group // each template is load only once
	templatesLock *sync.RWMutex       // provides atomic access to the templates field

	unlockFn git.RepositoryFsUnlockFn // unlocks underlying FS, called on free()
	freeLock *sync.RWMutex            // prevents cleanup of the repository while it is in use, see markInUse and free methods
}

// UnlockFn callback is returned by Manager. It must be called when the Cached Repository is no longer in use.
type UnlockFn func()

// newCachedRepository inits CachedRepository and preload all templates.
func newCachedRepository(ctx context.Context, d dependencies, gitRepo git.Repository, unlockFn git.RepositoryFsUnlockFn, tmplRepo *repository.Repository) *CachedRepository {
	r := &CachedRepository{
		d:             d,
		git:           gitRepo,
		repo:          tmplRepo,
		templates:     make(map[string]*template.Template),
		templatesInit: &singleflight.Group{},
		templatesLock: &sync.RWMutex{},
		unlockFn:      unlockFn,
		freeLock:      &sync.RWMutex{},
	}

	// Reload all templates, error in a template is logged
	_ = r.loadAllTemplates(ctx)
	return r
}

// Unwrap returns underlying repository.
func (r *CachedRepository) Unwrap() *repository.Repository {
	return r.repo
}

// String returns human-readable name of the repository with commit hash.
func (r *CachedRepository) String() string {
	commitHash := r.git.CommitHash()
	if commitHash != git.CommitHashNotSet {
		return fmt.Sprintf("%s:%s", r.repo.String(), r.git.CommitHash())
	} else {
		return r.repo.String()
	}
}

func (r *CachedRepository) UrlAndRef() string {
	def := r.repo.Definition()
	return fmt.Sprintf("%s:%s", def.Url, def.Ref)
}

// Hash returns unique identifier of the repository.
func (r *CachedRepository) Hash() string {
	return r.repo.Hash()
}

func (r *CachedRepository) Fs() filesystem.Fs {
	return r.repo.Fs()
}

func (r *CachedRepository) Template(ctx context.Context, reference model.TemplateRef) (*template.Template, error) {
	// The template must belong to the repository
	if reference.Repository().Hash() != r.Hash() {
		return nil, fmt.Errorf(`template "%s" is not from repository "%s"`, reference.FullName(), r.String())
	}

	// Check if is template already loaded
	name := reference.Name()
	r.templatesLock.RLock()
	value, found := r.templates[name] //nolint:ifshort
	r.templatesLock.RUnlock()
	if found {
		return value, nil
	}

	// Load template, there is used "single flight" library:
	// the function is called only once, but every caller will get the same results.
	ch := r.templatesInit.DoChan(name, func() (interface{}, error) {
		startTime := time.Now()
		r.d.Logger().Infof(`loading template "%s/%s"`, reference.FullName(), r.git.CommitHash())

		// Load template
		tmpl, err := loadTemplateOp.Run(ctx, r.d, r.repo, reference)
		if err != nil {
			return nil, fmt.Errorf(`cannot load template "%s": %w`, reference.FullName(), err)
		}

		// Cache value
		r.templatesLock.Lock()
		r.templates[name] = tmpl
		r.templatesLock.Unlock()

		// Load done
		r.d.Logger().Infof(`loaded template "%s/%s" | %s`, reference.FullName(), r.git.CommitHash(), time.Since(startTime))
		return tmpl, nil
	})

	// Check result
	result := <-ch
	if err := result.Err; err != nil {
		return nil, err
	}
	return result.Val.(*template.Template), nil
}

// update returns an updated copy of the repository if the repository has been changed.
func (r *CachedRepository) update(ctx context.Context) (*CachedRepository, bool, error) {
	// Only RemoteRepository can be updated
	if repo, ok := r.git.(*git.RemoteRepository); ok {
		// Log start
		startTime := time.Now()
		r.d.Logger().Infof(`repository "%s" update started`, r.UrlAndRef())

		// Pull
		result, err := pullOp.Run(ctx, repo, r.d)
		if err != nil {
			r.d.Logger().Errorf(`error while updating repository "%s": %w`, r.UrlAndRef(), err)
			return nil, false, err
		}

		// Done
		if result.Changed {
			r.d.Logger().Infof(`repository "%s" updated from %s to %s | %s`, r.UrlAndRef(), result.OldHash, result.NewHash, time.Since(startTime))
		} else {
			r.d.Logger().Infof(`repository "%s" update finished, no change found (%s) | %s`, r.UrlAndRef(), result.NewHash, time.Since(startTime))
		}

		// No change
		if !result.Changed {
			return r, false, nil
		}

		// Reload template repository
		fs, unlockFn := r.git.Fs()
		newData, err := loadRepositoryOp.Run(ctx, r.d, r.repo.Definition(), loadRepositoryOp.WithFs(fs))
		if err != nil {
			unlockFn()
			return nil, false, err
		}

		// Atomically exchange value, see markInUse method
		newRepo := newCachedRepository(ctx, r.d, r.git, unlockFn, newData)

		// Return new value
		return newRepo, true, nil
	}

	// No operation for a local repository
	return r, false, nil
}

// loadAllTemplates preloads all templates from the repository.
// If a template fails to load, the error is logged and also returned from this method.
func (r *CachedRepository) loadAllTemplates(ctx context.Context) error {
	startTime := time.Now()
	r.d.Logger().Infof(`loading all templates from repository "%s"`, r.String())

	wg := &sync.WaitGroup{}
	errors := utils.NewMultiError()
	for _, t := range r.repo.Templates() {
		t := t
		for _, v := range t.AllVersions() {
			v := v
			wg.Add(1)
			go func() {
				defer wg.Done()
				ref := model.NewTemplateRef(r.repo.Definition(), t.Id, v.Version.String())
				if _, err := r.Template(ctx, ref); err != nil {
					r.d.Logger().Errorf(`cannot load template "%s" from repository "%s": %s`, ref.FullName(), r.String(), err)
					errors.Append(fmt.Errorf(`cannot load template "%s": %w`, ref.Name(), err))
				}
			}()
		}
	}

	wg.Wait()
	if errors.Len() > 0 {
		r.d.Logger().Errorf(`cannot load all templates from repository "%s", see previous errors | %s`, r.String(), time.Since(startTime))
	} else {
		r.d.Logger().Infof(`loaded all templates from repository "%s" | %s`, r.String(), time.Since(startTime))
	}

	return errors.ErrorOrNil()
}

// markInUse is called when this repository starts to be used by a new request.
func (r *CachedRepository) markInUse() UnlockFn {
	// See Update method
	r.freeLock.RLock()
	return r.freeLock.RUnlock
}

// free is called when a new version of the repository is ready and the old one can be cleaned.
// It is waiting until all the requests that use this repository are finished.
func (r *CachedRepository) free() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		r.freeLock.Lock()
		defer r.freeLock.Unlock()
		r.unlockFn()
		r.d.Logger().Infof(`cleaned repository cache "%s"`, r.String())
		close(done)
	}()
	return done
}
