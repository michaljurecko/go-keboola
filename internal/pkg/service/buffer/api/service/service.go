package service

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/keboola/keboola-as-code/internal/pkg/idgenerator"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/api/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/api/gen/buffer"
	. "github.com/keboola/keboola-as-code/internal/pkg/service/buffer/api/gen/buffer"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/model"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/store/configstore"
	. "github.com/keboola/keboola-as-code/internal/pkg/service/common/httperror"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/strhelper"
)

type service struct{}

func New() Service {
	return &service{}
}

func (*service) APIRootIndex(dependencies.ForPublicRequest) (err error) {
	// Redirect / -> /v1
	return nil
}

func (*service) APIVersionIndex(dependencies.ForPublicRequest) (res *buffer.ServiceDetail, err error) {
	res = &ServiceDetail{
		API:           "buffer",
		Documentation: "https://buffer.keboola.com/v1/documentation",
	}
	return res, nil
}

func (*service) HealthCheck(dependencies.ForPublicRequest) (res string, err error) {
	return "OK", nil
}

func (*service) CreateReceiver(d dependencies.ForProjectRequest, payload *buffer.CreateReceiverPayload) (res *buffer.Receiver, err error) {
	ctx, store := d.RequestCtx(), d.ConfigStore()

	receiver := model.Receiver{
		ProjectID: d.ProjectID(),
		Name:      payload.Name,
	}

	// Generate receiver ID from Name if needed
	if payload.ReceiverID != nil && len(*payload.ReceiverID) != 0 {
		receiver.ID = *payload.ReceiverID
	} else {
		receiver.ID = strhelper.NormalizeName(receiver.Name)
	}

	// Generate Secret
	receiver.Secret = idgenerator.ReceiverSecret()

	// Persist receiver
	err = store.CreateReceiver(ctx, receiver)
	if err != nil {
		if errors.As(err, &configstore.LimitReachedError{}) {
			return nil, &GenericError{
				StatusCode: http.StatusUnprocessableEntity,
				Name:       "buffer.resourceLimitReached",
				Message:    fmt.Sprintf("Maximum number of receivers per project is %d.", configstore.MaxReceiversPerProject),
			}
		}
		if errors.As(err, &configstore.AlreadyExistsError{}) {
			return nil, &GenericError{
				StatusCode: http.StatusConflict,
				Name:       "buffer.alreadyExists",
				Message:    fmt.Sprintf(`Receiver "%s" already exists.`, receiver.ID),
			}
		}
		// nolint:godox
		// TODO: maybe handle validation error
		return nil, errors.Wrapf(err, "failed to create receiver \"%s\"", receiver.ID)
	}

	for _, exportData := range payload.Exports {
		export := model.Export{
			Name: exportData.Name,
			ImportConditions: model.ImportConditions{
				Count: exportData.Conditions.Count,
				Size:  datasize.ByteSize(exportData.Conditions.Size),
				Time:  time.Duration(exportData.Conditions.Time),
			},
		}

		// Generate export ID from Name if needed
		if exportData.ExportID != nil && len(*exportData.ExportID) != 0 {
			export.ID = *exportData.ExportID
		} else {
			export.ID = strhelper.NormalizeName(export.Name)
		}

		// Persist export
		err = store.CreateExport(ctx, receiver.ProjectID, receiver.ID, export)
		if err != nil {
			if errors.As(err, &configstore.LimitReachedError{}) {
				return nil, &GenericError{
					StatusCode: http.StatusUnprocessableEntity,
					Name:       "buffer.resourceLimitReached",
					Message:    fmt.Sprintf("Maximum number of exports per receiver is %d.", configstore.MaxExportsPerReceiver),
				}
			}
			if errors.As(err, &configstore.AlreadyExistsError{}) {
				return nil, &GenericError{
					StatusCode: http.StatusConflict,
					Name:       "buffer.alreadyExists",
					Message:    fmt.Sprintf(`Export "%s" already exists.`, export.ID),
				}
			}
			// nolint:godox
			// TODO: maybe handle validation error
			return nil, err
		}

		// nolint:godox
		// TODO: create mappings
	}

	url := formatUrl(d.BufferApiHost(), receiver.ProjectID, receiver.ID, receiver.Secret)
	resp := &buffer.Receiver{
		ReceiverID: &receiver.ID,
		Name:       &receiver.Name,
		URL:        &url,
		Exports:    []*Export{},
	}

	return resp, nil
}

func (*service) GetReceiver(d dependencies.ForProjectRequest, payload *buffer.GetReceiverPayload) (res *buffer.Receiver, err error) {
	ctx, store := d.RequestCtx(), d.ConfigStore()

	projectID, receiverID := d.ProjectID(), payload.ReceiverID

	receiver, err := store.GetReceiver(ctx, projectID, receiverID)
	if err != nil {
		if errors.As(err, &configstore.NotFoundError{}) {
			return nil, &GenericError{
				StatusCode: http.StatusNotFound,
				Name:       "buffer.receiverNotFound",
				Message:    fmt.Sprintf("Receiver \"%s\" not found", receiverID),
			}
		}
		return nil, errors.Wrapf(err, "failed to get receiver \"%s\" in project \"%d\"", receiverID, projectID)
	}

	// nolint: godox
	// TODO: get exports

	url := formatUrl(d.BufferApiHost(), receiver.ProjectID, receiver.ID, receiver.Secret)
	resp := &buffer.Receiver{
		ReceiverID: &receiver.ID,
		Name:       &receiver.Name,
		URL:        &url,
		Exports:    []*Export{},
	}

	return resp, nil
}

func (*service) ListReceivers(d dependencies.ForProjectRequest, _ *buffer.ListReceiversPayload) (res *buffer.ListReceiversResult, err error) {
	ctx, store := d.RequestCtx(), d.ConfigStore()

	projectID := d.ProjectID()

	data, err := store.ListReceivers(ctx, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list receivers in project \"%d\"", projectID)
	}

	bufferApiHost := d.BufferApiHost()

	receivers := make([]*buffer.Receiver, 0, len(data))
	for _, entry := range data {
		url := formatUrl(bufferApiHost, entry.ProjectID, entry.ID, entry.Secret)
		receivers = append(receivers, &buffer.Receiver{
			ReceiverID: &entry.ID,
			Name:       &entry.Name,
			URL:        &url,
			Exports:    []*Export{},
		})
	}

	sort.SliceStable(receivers, func(i, j int) bool {
		return *receivers[i].ReceiverID < *receivers[j].ReceiverID
	})

	return &buffer.ListReceiversResult{Receivers: receivers}, nil
}

func (*service) DeleteReceiver(d dependencies.ForProjectRequest, payload *buffer.DeleteReceiverPayload) (err error) {
	ctx, store := d.RequestCtx(), d.ConfigStore()

	projectID, receiverID := d.ProjectID(), payload.ReceiverID

	err = store.DeleteReceiver(ctx, projectID, receiverID)
	if err != nil {
		if errors.As(err, &configstore.NotFoundError{}) {
			return &GenericError{
				StatusCode: 404,
				Name:       "buffer.receiverNotFound",
				Message:    fmt.Sprintf("Receiver \"%s\" not found", receiverID),
			}
		}
		return err
	}

	return nil
}

func (*service) RefreshReceiverTokens(dependencies.ForProjectRequest, *buffer.RefreshReceiverTokensPayload) (res *buffer.Receiver, err error) {
	return nil, &NotImplementedError{}
}

func (*service) CreateExport(dependencies.ForProjectRequest, *buffer.CreateExportPayload) (res *buffer.Receiver, err error) {
	return nil, &NotImplementedError{}
}

func (*service) UpdateExport(dependencies.ForProjectRequest, *buffer.UpdateExportPayload) (res *buffer.Receiver, err error) {
	return nil, &NotImplementedError{}
}

func (*service) DeleteExport(dependencies.ForProjectRequest, *buffer.DeleteExportPayload) (res *buffer.Receiver, err error) {
	return nil, &NotImplementedError{}
}

func (*service) Import(dependencies.ForPublicRequest, *buffer.ImportPayload, io.ReadCloser) (err error) {
	return &NotImplementedError{}
}

func formatUrl(bufferApiHost string, projectID int, receiverID string, secret string) string {
	return fmt.Sprintf("https://%s/v1/import/%d/%s/#/%s", bufferApiHost, projectID, receiverID, secret)
}
