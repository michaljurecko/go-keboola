package storageapi

import (
	"sort"
	"sync"

	"github.com/go-resty/resty/v2"

	client2 "github.com/keboola/keboola-as-code/internal/pkg/http/client"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
)

// TicketProvider generates new IDs and GUARANTEES that the IDs will be in the same order as the Request method was called.
type TicketProvider struct {
	lock      *sync.Mutex
	api       *Api
	pool      *client2.Pool
	tickets   []*model.Ticket
	callbacks []func(ticket *model.Ticket)
}

func (a *Api) NewTicketProvider() *TicketProvider {
	return &TicketProvider{lock: &sync.Mutex{}, api: a, pool: a.NewPool()}
}

func (t *TicketProvider) Request(onSuccess func(ticket *model.Ticket)) {
	t.callbacks = append(t.callbacks, onSuccess)
	t.pool.
		Request(t.api.GenerateNewIdRequest()).
		OnSuccess(func(response *client2.Response) {
			t.lock.Lock()
			defer t.lock.Unlock()
			ticket := response.Result().(*model.Ticket)
			t.tickets = append(t.tickets, ticket)
		}).
		Send()
}

func (t *TicketProvider) Resolve() error {
	if err := t.pool.StartAndWait(); err != nil {
		return err
	}

	sort.SliceStable(t.tickets, func(i, j int) bool {
		return t.tickets[i].Id < t.tickets[j].Id
	})

	for index, ticket := range t.tickets {
		t.callbacks[index](ticket)
	}

	return nil
}

func (a *Api) GenerateNewId() (*model.Ticket, error) {
	response := a.GenerateNewIdRequest().Send().Response
	if response.HasResult() {
		return response.Result().(*model.Ticket), nil
	}
	return nil, response.Err()
}

// GenerateNewIdRequest https://keboola.docs.apiary.io/#reference/tickets/generate-unique-id/generate-new-id
func (a *Api) GenerateNewIdRequest() *client2.Request {
	ticket := &model.Ticket{}
	return a.
		NewRequest(resty.MethodPost, "tickets").
		SetResult(ticket)
}
