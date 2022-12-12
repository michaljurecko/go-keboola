package schema

import (
	"strconv"
	"time"

	storeKey "github.com/keboola/keboola-as-code/internal/pkg/service/buffer/store/key"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/store/model"
	. "github.com/keboola/keboola-as-code/internal/pkg/service/common/etcdop"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

type slices = PrefixT[model.Slice]

type Slices struct {
	schema *Schema
	slices
}

type SlicesInAState struct {
	slices
}

type SlicesInFile struct {
	slices
}

func (v *Schema) Slices() Slices {
	return Slices{schema: v, slices: NewTypedPrefix[model.Slice](
		NewPrefix("slice"),
		v.serde,
	)}
}

func (v Slices) Opened() SlicesInAState {
	return SlicesInAState{slices: v.slices.Add("opened")}
}

func (v Slices) Closing() SlicesInAState {
	return SlicesInAState{slices: v.slices.Add("closing")}
}

func (v Slices) Closed() SlicesInAState {
	return SlicesInAState{slices: v.slices.Add("closed")}
}

func (v Slices) Uploaded() SlicesInAState {
	return SlicesInAState{slices: v.slices.Add("uploaded")}
}

func (v Slices) Failed() SlicesInAState {
	return SlicesInAState{slices: v.slices.Add("failed")}
}

func (v SlicesInAState) ByKey(k storeKey.SliceKey) KeyT[model.Slice] {
	return v.InFile(k.FileKey).ID(k.SliceID)
}

func (v SlicesInAState) InFile(k storeKey.FileKey) SlicesInFile {
	if k.ProjectID == 0 {
		panic(errors.New("slice projectID cannot be empty"))
	}
	if k.ReceiverID == "" {
		panic(errors.New("slice receiverID cannot be empty"))
	}
	if k.ExportID == "" {
		panic(errors.New("slice exportID cannot be empty"))
	}
	if k.FileID.IsZero() {
		panic(errors.New("slice fileID cannot be empty"))
	}
	return SlicesInFile{slices: v.slices.Add(strconv.Itoa(k.ProjectID)).Add(k.ReceiverID).Add(k.ExportID).Add(storeKey.FormatTime(k.FileID))}
}

func (v SlicesInFile) ID(sliceID time.Time) KeyT[model.Slice] {
	if sliceID.IsZero() {
		panic(errors.New("slice sliceID cannot be empty"))
	}
	return v.slices.Key(storeKey.FormatTime(sliceID))
}
