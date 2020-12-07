package api

import (
	"fmt"
	"github.com/pinpt/agent/v4/sdk"
	"net/http"
	"net/url"
)

// Requester2 requester
type Requester2 struct {
	concurrency chan bool
	client      sdk.HTTPClient
}

func NewRequester2(client sdk.HTTPClient, concurrency int) *Requester2 {
	return &Requester2{
		client:      client,
		concurrency: make(chan bool, concurrency),
	}
}

// Get request
func (r *Requester2) Get(logger sdk.Logger, endpoint string, params url.Values, response interface{}) (np NextPage, err error) {
	r.concurrency <- true
	defer func() {
		<-r.concurrency
	}()

	ir := internalRequest{
		EndPoint:    endpoint,
		Params:      params,
		Response:    &response,
		RequestType: Get,
		logger: logger,
	}

	return r.makeRequestRetry(&ir, 0)

}

func (r *Requester2) makeRequestRetry(req *internalRequest, generalRetry int) (np NextPage, err error) {
	var isRetryable bool
	isRetryable, np, err = r.request(req, generalRetry+1)
	if err != nil {
		if !isRetryable {
			return np, err
		}
		if generalRetry >= maxGeneralRetries {
			return np, fmt.Errorf(`can't retry request, too many retries, err: %v`, err)
		}
		return r.makeRequestRetry(req, generalRetry+1)
	}
	return
}

func (e *Requester2) request(r *internalRequest, retryThrottled int) (isErrorRetryable bool, np NextPage, rerr error) {

	headers := sdk.WithHTTPHeader("Accept", "application/json")
	endpoint := sdk.WithEndpoint(r.EndPoint)
	parameters := sdk.WithGetQueryParameters(r.Params)

	sdk.LogDebug(r.logger, "request info", "method", r.RequestType, "endpoint", r.EndPoint, "parameters", r.Params)

	var resp *sdk.HTTPResponse
	switch r.RequestType {
	case Get:
		resp, rerr = e.client.Get(&r.Response, headers, endpoint, parameters)
		if rerr != nil {
			return true, NextPage(resp.Headers.Get("X-Next-Page")), fmt.Errorf("error on get: %s %s", rerr, string(resp.Body))
		}
	case Post:
		reader, err := r.getDataReader()
		if err != nil {
			sdk.LogDebug(r.logger, "request response", "resp", string(resp.Body))
			return true, NextPage(resp.Headers.Get("X-Next-Page")), fmt.Errorf("error on post: %s", err)
		}
		resp, rerr = e.client.Post(reader, &r.Response, headers, endpoint, parameters)
		if rerr != nil {
			return true, NextPage(resp.Headers.Get("X-Next-Page")), fmt.Errorf("error on post: %s %s", rerr, string(resp.Body))
		}
	case Delete:
		resp, rerr = e.client.Delete(&r.Response, headers, endpoint, parameters)
		if rerr != nil {
			return true, NextPage(resp.Headers.Get("X-Next-Page")), fmt.Errorf("error on delete: %s %s", rerr, string(resp.Body))
		}
	case Put:
		reader, err := r.getDataReader()
		if err != nil {
			sdk.LogDebug(r.logger, "request response", "resp", string(resp.Body))
			return true, np, fmt.Errorf("error on put: %s", err)
		}
		resp, rerr = e.client.Put(reader, &r.Response, headers, endpoint, parameters)
		if rerr != nil {
			return true, NextPage(resp.Headers.Get("X-Next-Page")), fmt.Errorf("error on put: %s %s", rerr, string(resp.Body))
		}
	}

	rateLimited := func() (isErrorRetryable bool, NextPage NextPage, rerr error) {

		rateLimit(r.logger, retryThrottled)

		return true, np, fmt.Errorf("too many requests")

	}

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusNoContent {

		if resp.StatusCode == http.StatusTooManyRequests {
			return rateLimited()
		}

		if resp.StatusCode == http.StatusForbidden {

			return false, np, fmt.Errorf("permissions error")
		}

		sdk.LogWarn(r.logger, "gitlab returned invalid status code, retrying", "code", resp.StatusCode, "retry", retryThrottled)

		return true, NextPage(resp.Headers.Get("X-Next-Page")), fmt.Errorf("request with status %d", resp.StatusCode)
	}

	return false, NextPage(resp.Headers.Get("X-Next-Page")), nil
}