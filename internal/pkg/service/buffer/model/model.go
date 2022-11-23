package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/c2h5oh/datasize"

	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/model/column"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

const (
	TableStageIn  = "in"
	TableStageOut = "out"
	TableStageSys = "sys"
)

type TableID struct {
	Stage  string `json:"stage" validate:"required,oneof=in out sys"`
	Bucket string `json:"bucketName" validate:"required,min=1,max=96"`
	Table  string `json:"tableName" validate:"required,min=1,max=96"`
}

func (t TableID) String() string {
	return fmt.Sprintf("%s.c-%s.%s", t.Stage, t.Bucket, t.Table)
}

// nolint:gochecknoglobals
var tableStagesMap = map[string]bool{
	TableStageIn:  true,
	TableStageOut: true,
	TableStageSys: true,
}

func ParseTableID(v string) (TableID, error) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return TableID{}, errors.Errorf(`invalid table ID "%s"`, v)
	}

	stage, bucket, table := parts[0], parts[1], parts[2]

	if !tableStagesMap[stage] {
		return TableID{}, errors.Errorf(`invalid table ID "%s"`, v)
	}

	if !strings.HasPrefix(bucket, "c-") {
		return TableID{}, errors.Errorf(`invalid table ID "%s"`, v)
	}
	bucket = strings.TrimPrefix(bucket, "c-")

	id := TableID{
		Stage:  stage,
		Bucket: bucket,
		Table:  table,
	}

	return id, nil
}

type Mapping struct {
	RevisionID  int            `json:"revisionId" validate:"max=100"`
	TableID     TableID        `json:"tableId" validate:"required"`
	Incremental bool           `json:"incremental"`
	Columns     column.Columns `json:"columns" validate:"required,min=1,max=50"`
}

type Receiver struct {
	ID        string `json:"receiverId" validate:"required,min=1,max=48"`
	ProjectID int    `json:"projectId"`
	Name      string `json:"name" validate:"required,min=1,max=40"`
	Secret    string `json:"secret" validate:"required,len=48"`
}

type ImportConditions struct {
	Count int               `json:"count" validate:"min=1,max=10000000"`
	Size  datasize.ByteSize `json:"size" validate:"min=100,max=50000000"`
	Time  time.Duration     `json:"time" validate:"min=30000000000,max=86400000000000"`
}

type Export struct {
	ID               string           `json:"exportId" validate:"required,min=1,max=48"`
	Name             string           `json:"name" validate:"required,min=1,max=40"`
	ImportConditions ImportConditions `json:"importConditions" validate:"required"`
}
