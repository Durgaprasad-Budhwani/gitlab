package api

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pinpt/agent.next/sdk"
)

// RequesterOpts requester opts
type RequesterOpts struct {
	Logger      sdk.Logger
	Concurrency chan bool
	Client      sdk.HTTPClient
}

// NewRequester new requester
func NewRequester(opts *RequesterOpts) *Requester {
	re := &Requester{}

	re.opts = opts

	return re
}

type internalRequest struct {
	EndPoint string
	Params   url.Values
	Response interface{}
	NextPage NextPage
}

// Requester requester
type Requester struct {
	opts *RequesterOpts
}

// MakeRequest make request
func (e *Requester) MakeRequest(endpoint string, params url.Values, response interface{}) (np NextPage, err error) {
	e.opts.Concurrency <- true
	defer func() {
		<-e.opts.Concurrency
	}()

	ir := internalRequest{
		EndPoint: endpoint,
		Response: &response,
		Params:   params,
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

type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"message"`
}

func (e *Requester) request(r *internalRequest, retryThrottled int) (isErrorRetryable bool, np NextPage, rerr error) {

	headers := sdk.WithHTTPHeader("Accept", "application/json")
	endpoint := sdk.WithEndpoint(r.EndPoint)
	parameters := sdk.WithGetQueryParameters(r.Params)

	resp, err := e.opts.Client.Get(&r.Response, headers, endpoint, parameters)
	if err != nil {
		return true, np, err
	}

	rateLimited := func() (isErrorRetryable bool, NextPage NextPage, rerr error) {

		waitTime := time.Minute * 3

		sdk.LogWarn(e.opts.Logger, "api request failed due to throttling, the quota of 600 calls has been reached, will sleep for 3m and retry", "retryThrottled", retryThrottled)

		paused := time.Now()
		resumeDate := paused.Add(waitTime)
		sdk.LogWarn(e.opts.Logger, "gitlab paused, it will resume in %v, resume data %v", waitTime, resumeDate)

		time.Sleep(waitTime)

		sdk.LogWarn(e.opts.Logger, fmt.Sprintf("gitlab resumed, time elapsed %v", time.Since(paused)))

		return true, np, fmt.Errorf("too many requests")

	}

	if resp.StatusCode != http.StatusOK {

		if resp.StatusCode == http.StatusTooManyRequests {
			return rateLimited()
		}

		if resp.StatusCode == http.StatusForbidden {

			return false, np, fmt.Errorf("permissions error")
		}

		sdk.LogWarn(e.opts.Logger, "gitlab returned invalid status code, retrying", "code", resp.StatusCode, "retry", retryThrottled)

		return true, np, fmt.Errorf("request with status %d", resp.StatusCode)
	}

	return false, NextPage(resp.Headers.Get("X-Next-Page")), nil
}
