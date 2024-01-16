package repository

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/sink/tablesink/storage"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/etcdop/serde"
)

func TestVolumeSchema(t *testing.T) {
	t.Parallel()
	s := newVolumeSchema(serde.NewJSON(serde.NoValidation))

	volumeID := storage.VolumeID("my-volume")

	cases := []struct{ actual, expected string }{
		{
			s.Prefix(),
			"storage/volume/",
		},
		{
			s.WriterVolumes().Prefix(),
			"storage/volume/writer/",
		},
		{
			s.ReaderVolumes().Prefix(),
			"storage/volume/reader/",
		},
		{
			s.WriterVolume(volumeID).Key(),
			"storage/volume/writer/my-volume",
		},
		{
			s.ReaderVolume(volumeID).Key(),
			"storage/volume/reader/my-volume",
		},
	}

	for i, c := range cases {
		assert.Equal(t, c.expected, c.actual, fmt.Sprintf(`case "%d"`, i+1))
	}
}
