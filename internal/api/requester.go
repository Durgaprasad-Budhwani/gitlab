package api

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pinpt/agent.next/sdk"
)

// Requester requester
type Requester struct {
	logger      sdk.Logger
	concurrency chan bool
	client      sdk.HTTPClient
}

// NewRequester new requester
func NewRequester(logger sdk.Logger, client sdk.HTTPClient, concurrency int) *Requester {
	return &Requester{
		logger:      logger,
		client:      client,
		concurrency: make(chan bool, concurrency),
	}
}

type internalRequest struct {
	EndPoint string
	Params   url.Values
	Response interface{}
}

// MakeRequest make request
func (e *Requester) MakeRequest(endpoint string, params url.Values, response interface{}) (np NextPage, err error) {
	e.concurrency <- true
	defer func() {
		<-e.concurrency
	}()

	ir := internalRequest{
		EndPoint: endpoint,
		Params:   params,
		Response: &response,
	}

	return e.makeRequestRetry(&ir, 0)

}

const maxGeneralRetries = 2

func (e *Requester) makeRequestRetry(req *internalRequest, generalRetry int) (np NextPage, err error) {
	var isRetryable bool
	isRetryable, np, err = e.request(req, generalRetry+1)
	if err != nil {
		if !isRetryable {
			return np, err
		}
		if generalRetry >= maxGeneralRetries {
			return np, fmt.Errorf(`can't retry request, too many retries, err: %v`, err)
		}
		return e.makeRequestRetry(req, generalRetry+1)
	}
	return
}

const maxThrottledRetries = 3

func (e *Requester) request(r *internalRequest, retryThrottled int) (isErrorRetryable bool, np NextPage, rerr error) {

	headers := sdk.WithHTTPHeader("Accept", "application/json")
	endpoint := sdk.WithEndpoint(r.EndPoint)
	parameters := sdk.WithGetQueryParameters(r.Params)

	resp, err := e.client.Get(&r.Response, headers, endpoint, parameters)
	if err != nil {
		return true, np, err
	}

	rateLimited := func() (isErrorRetryable bool, NextPage NextPage, rerr error) {

		waitTime := time.Minute * 3

		sdk.LogWarn(e.logger, "api request failed due to throttling, the quota of 600 calls has been reached, will sleep for 3m and retry", "retryThrottled", retryThrottled)

		paused := time.Now()
		resumeDate := paused.Add(waitTime)
		sdk.LogWarn(e.logger, "gitlab paused, it will resume in %v, resume data %v", waitTime, resumeDate)

		time.Sleep(waitTime)

		sdk.LogWarn(e.logger, fmt.Sprintf("gitlab resumed, time elapsed %v", time.Since(paused)))

		return true, np, fmt.Errorf("too many requests")

	}

	if resp.StatusCode != http.StatusOK {

		if resp.StatusCode == http.StatusTooManyRequests {
			return rateLimited()
		}

		if resp.StatusCode == http.StatusForbidden {

			return false, np, fmt.Errorf("permissions error")
		}

		sdk.LogWarn(e.logger, "gitlab returned invalid status code, retrying", "code", resp.StatusCode, "retry", retryThrottled)

		return true, np, fmt.Errorf("request with status %d", resp.StatusCode)
	}

	return false, NextPage(resp.Headers.Get("X-Next-Page")), nil
}
