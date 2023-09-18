package volume

import (
	"os"
	"sort"

	"github.com/keboola/keboola-as-code/internal/pkg/filesystem"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/storage"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/storage/level/local/writer"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/storage/level/local/writer/writechain"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

const (
	sliceDirPerm = 0o750
)

type writerRef struct {
	writer.Writer
}

func (v *Volume) NewWriterFor(slice *storage.Slice) (out *writer.EventWriter, err error) {
	// Check context
	if err := v.ctx.Err(); err != nil {
		return nil, errors.Errorf(`writer for slice "%s cannot be created, volume is closed: %w`, slice.SliceKey.String(), err)
	}

	// Setup logger
	logger := v.logger

	// Setup events
	events := v.events.Clone()

	// Check if the writer already exists, if not, register an empty reference to unlock immediately
	ref, exists := v.addWriter(slice.SliceKey)
	if exists {
		return nil, errors.Errorf(`writer for slice "%s" already exists`, slice.SliceKey.String())
	}

	// Register writer close callback
	events.OnWriterClose(func(_ writer.Writer, _ error) error {
		v.removeWriter(slice.SliceKey)
		return nil
	})

	// Close resources on a creation error
	var file File
	var chain *writechain.Chain
	defer func() {
		if err == nil {
			// Update reference
			ref.Writer = out
		} else {
			// Close resources
			if chain != nil {
				_ = chain.Close()
			} else if file != nil {
				_ = file.Close()
			}
			// Unregister the writer
			v.removeWriter(slice.SliceKey)
		}
	}()

	// Create directory if not exists
	dirPath := filesystem.Join(v.Path(), slice.LocalStorage.Dir)
	if err = os.Mkdir(dirPath, sliceDirPerm); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, errors.Errorf(`cannot create slice directory "%s": %w`, dirPath, err)
	}

	// Open file
	filePath := filesystem.Join(dirPath, slice.LocalStorage.Filename)
	file, err = v.config.fileOpener(filePath)
	if err == nil {
		logger.Debug("opened file")
	} else {
		logger.Error(`cannot open file "%s": %s`, filePath, err)
		return nil, err
	}

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Allocate disk space
	if isNew := stat.Size() == 0; isNew {
		if size := slice.LocalStorage.AllocateSpace; size != 0 {
			if ok, err := v.config.allocator.Allocate(file, size); ok {
				logger.Debugf(`allocated disk space "%s"`, size)
			} else if err != nil {
				// The error is not fatal
				logger.Errorf(`cannot allocate disk space "%s", allocation skipped: %s`, size, err)
			} else {
				logger.Debug("disk space allocation is not supported")
			}
		} else {
			logger.Debug("disk space allocation is disabled")
		}
	}

	// Init writers chain
	chain = writechain.New(logger, file)

	// Create writer via factory
	w, err := v.config.writerFactory(writer.NewBaseWriter(logger, v.clock, slice, dirPath, filePath, chain, events))
	if err != nil {
		return nil, err
	}

	// Wrap the writer to add events dispatching
	return writer.NewEventWriter(w, events)
}

func (v *Volume) Writers() (out []writer.Writer) {
	v.writersLock.Lock()
	defer v.writersLock.Unlock()

	out = make([]writer.Writer, 0, len(v.writers))
	for _, w := range v.writers {
		out = append(out, w)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].SliceKey().String() < out[j].SliceKey().String()
	})

	return out
}

func (v *Volume) addWriter(k storage.SliceKey) (ref *writerRef, exists bool) {
	v.writersLock.Lock()
	defer v.writersLock.Unlock()

	key := k.String()
	ref, exists = v.writers[key]
	if !exists {
		// Register a new empty reference, it will be initialized later.
		// Empty reference is used to make possible to create multiple writers without being blocked by the lock.
		ref = &writerRef{}
		v.writers[key] = ref
	}

	return ref, exists
}

func (v *Volume) removeWriter(k storage.SliceKey) {
	v.writersLock.Lock()
	defer v.writersLock.Unlock()
	delete(v.writers, k.String())
}
