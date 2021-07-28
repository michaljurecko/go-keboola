package encryption

import (
	"context"
	"fmt"
	"keboola-as-code/src/client"
	"strings"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type EncryptionApi struct {
	apiHost string
	client  *client.Client
	logger  *zap.SugaredLogger
}

// Error represents Encryption API error structure
type Error struct {
	Message     string `json:"error"`
	ErrCode     int    `json:"code"`
	ExceptionId string `json:"exceptionId"`
}

func (e *Error) Error() string {
	msg := fmt.Sprintf(`"%v", errCode: "%v"`, e.Message, e.ErrCode)
	if len(e.ExceptionId) > 0 {
		msg += fmt.Sprintf(`, exceptionId: "%s"`, e.ExceptionId)
	}
	return msg
}

func getEncryptionApiHost(connectionApiHost string) string {
	return strings.ReplaceAll(connectionApiHost, "connection.", "encryption.")
}

func NewEncryptionApi(connectionApiHost string, ctx context.Context, logger *zap.SugaredLogger, verbose bool) *EncryptionApi {
	apiHost := getEncryptionApiHost(connectionApiHost)
	apiHostUrl := "https://" + apiHost
	c := client.NewClient(ctx, logger, verbose).WithHostUrl(apiHostUrl)
	c.SetError(&Error{})
	api := &EncryptionApi{client: c, logger: logger, apiHost: apiHost}
	return api
}

func (a *EncryptionApi) NewRequest(method string, url string) *client.Request {
	return a.client.NewRequest(method, url)
}

func (a *EncryptionApi) createRequest(componentId string, projectId string, requestBody map[string]string) (*client.Request, error) {
	// Create request
	result := make(map[string]string)
	request := a.
		client.NewRequest(resty.MethodPost, "encrypt").
		SetQueryParam("componentId", componentId).
		SetQueryParam("projectId", projectId).
		SetResult(&result)
	request.Request.SetBody(requestBody)
	request.Request.SetHeader("Content-Type", "application/json")

	return request, nil
}

func (a *EncryptionApi) EncryptMapValues(componentId string, projectId string, mapValues map[string]string) (map[string]string, error) {
	request, err := a.createRequest(componentId, projectId, mapValues)
	if err != nil {
		return nil, err
	}
	response := request.Send().Response
	if response.HasResult() {
		return *response.Result().(*map[string]string), nil
	}
	return nil, response.Err()
}
