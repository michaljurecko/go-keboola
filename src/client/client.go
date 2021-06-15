package client

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"keboola-as-code/src/utils"
	"keboola-as-code/src/version"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	RequestTimeout   = 45 * time.Second
	HttpTimeout      = 30 * time.Second
	IdleConnTimeout  = 90 * time.Second
	KeepAlive        = 30 * time.Second
	MaxIdleConns     = 64
	RetryCount       = 5
	RetryWaitTime    = 100 * time.Millisecond
	RetryWaitTimeMax = 3 * time.Second
)

type Client struct {
	parentCtx        context.Context // context for parallel execution
	logger           *Logger
	resty            *resty.Client
	requestIdCounter *utils.SafeCounter
}

func NewClient(ctx context.Context, logger *zap.SugaredLogger, verbose bool) *Client {
	client := &Client{}
	client.logger = &Logger{logger}
	client.parentCtx = ctx
	client.resty = createHttpClient(client.logger)
	client.requestIdCounter = utils.NewSafeCounter(0)
	setupLogs(client, verbose)
	return client
}

func (c Client) WithHostUrl(hostUrl string) *Client {
	c.resty.SetHostURL(hostUrl)
	return &c
}

func (c *Client) Send(request *Request) {
	restyResponse, err := request.RestyRequest().Send()
	request.response = NewResponse(request, restyResponse, err)
	request.invokeListeners()
}

func (c *Client) Request(request *Request) *Request {
	request.sender = c
	return request
}

func (c *Client) NewRequest(method string, url string) *Request {
	r := c.resty.R()
	r.Method = method
	r.URL = url
	return NewRequest(c.requestIdCounter.IncAndGet(), c, r)
}

func (c *Client) HostUrl() string {
	return c.resty.HostURL
}

func (c *Client) SetHeader(header, value string) *Client {
	c.resty.SetHeader(header, value)
	return c
}

func (c *Client) Header() http.Header {
	return c.resty.Header
}

func (c *Client) SetError(err interface{}) *Client {
	c.resty.SetError(err)
	return c
}

func (c *Client) SetRetry(count int, waitTime time.Duration, maxWaitTime time.Duration) {
	c.resty.RetryWaitTime = waitTime
	c.resty.RetryMaxWaitTime = maxWaitTime
	c.resty.RetryCount = count
}

func createHttpClient(logger *Logger) *resty.Client {
	r := resty.New()
	r.SetLogger(logger)
	r.SetHeader("User-Agent", fmt.Sprintf("keboola-as-code/%s", version.BuildVersion))
	r.SetTimeout(RequestTimeout)
	r.SetTransport(createTransport())
	r.SetRetryCount(RetryCount)
	r.SetRetryWaitTime(RetryWaitTime)
	r.SetRetryMaxWaitTime(RetryWaitTimeMax)
	r.AddRetryCondition(func(response *resty.Response, err error) bool {
		// On network errors
		if err != nil && response == nil {
			switch true {
			case
				strings.Contains(err.Error(), "No address associated with hostname"):
				return false
			default:
				return true
			}
		}

		// On status codes
		switch response.StatusCode() {
		case
			http.StatusRequestTimeout,
			http.StatusConflict,
			http.StatusLocked,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	})

	return r
}

func createTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   HttpTimeout,
		KeepAlive: KeepAlive,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          MaxIdleConns,
		IdleConnTimeout:       IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   MaxIdleConns,
	}
}

func setupLogs(client *Client, verbose bool) {
	// Debug full request and response if verbose = true
	// Secrets are hidden see Logger
	if verbose {
		client.resty.SetDebug(true)
		client.resty.SetDebugBodyLimit(2 * 1024)
	}

	// Log each request when done
	client.resty.OnAfterResponse(func(c *resty.Client, res *resty.Response) error {
		req := res.Request
		msg := responseToLog(res)
		if res.IsSuccess() {
			// Log success
			client.logger.Debugf("%s", msg)
		}

		// Return error if present
		err := res.Error()
		if err != nil {
			// Set response to error if supported
			if v, ok := err.(ErrorWithResponse); ok {
				v.SetResponse(res)
			}

			// Return err, wrap if needed
			if v, ok := err.(error); ok {
				return v
			} else {
				return fmt.Errorf("url:\"%s\", error: \"%s\"", req.URL, err)
			}
		}

		// Return error if request failed
		if res.IsError() {
			return fmt.Errorf(`%s "%s" returned http code %d`, req.Method, req.URL, res.StatusCode())
		}

		return nil
	})
}

func responseToLog(res *resty.Response) string {
	req := res.Request
	return fmt.Sprintf("%s %s | %d | %s", req.Method, req.URL, res.StatusCode(), res.Time())
}
