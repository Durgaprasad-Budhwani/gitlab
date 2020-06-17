package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/dnaeon/go-vcr/recorder"
	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/httpdefaults"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

// RequesterOpts requester opts
type RequesterOpts struct {
	Logger sdk.Logger
	APIURL string
	APIKey string
	// AccessToken        string
	InsecureSkipVerify bool
	// ServerType         ServerType
	// Concurrency        chan bool
	client      *http.Client
	UseRecorder bool // used for development only
}

// NewRequester new requester
func NewRequester(opts RequesterOpts) *Requester {
	re := &Requester{}
	{

		c := &http.Client{}
		transport := httpdefaults.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify}
		c.Transport = transport

		opts.client = c
	}

	re.opts = opts

	return re
}

type internalRequest struct {
	URL      string
	Params   url.Values
	Response interface{}
	PageInfo PageInfo
}

type errorState struct {
	sync.Mutex
	err error
}

func (e *errorState) setError(err error) {
	e.Lock()
	defer e.Unlock()
	e.err = err
}

func (e *errorState) getError() error {
	e.Lock()
	defer e.Unlock()
	return e.err
}

// Requester requester
type Requester struct {
	opts RequesterOpts
}

// MakeRequest make request
func (e *Requester) MakeRequest(url string, params url.Values, response interface{}) (pi PageInfo, err error) {
	// TODO: Uncomment once we add concurrency
	// e.opts.Concurrency <- true
	// defer func() {
	// 	<-e.opts.Concurrency
	// }()

	ir := internalRequest{
		URL:      url,
		Response: &response,
		Params:   params,
	}

	return e.makeRequestRetry(&ir, 0)

}

const maxGeneralRetries = 2

func (e *Requester) makeRequestRetry(req *internalRequest, generalRetry int) (pageInfo PageInfo, err error) {
	var isRetryable bool
	isRetryable, pageInfo, err = e.request(req, generalRetry+1)
	if err != nil {
		if !isRetryable {
			return pageInfo, err
		}
		if generalRetry >= maxGeneralRetries {
			return pageInfo, fmt.Errorf(`can't retry request, too many retries, err: %v`, err)
		}
		return e.makeRequestRetry(req, generalRetry+1)
	}
	return
}

func (e *Requester) setAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "bearer "+e.opts.APIKey)
	// No need to set "Private-Token" header with apikey.
}

const maxThrottledRetries = 3

type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (e *Requester) request(r *internalRequest, retryThrottled int) (isErrorRetryable bool, pi PageInfo, rerr error) {
	u := pstrings.JoinURL(e.opts.APIURL, r.URL)

	if len(r.Params) != 0 {
		u += "?" + r.Params.Encode()
	}

	if e.opts.UseRecorder {
		rr, err := recorder.New("fixtures/" + u)
		if err != nil {
			rerr = err
			return
		}
		defer rr.Stop()

		e.opts.client.Transport = rr
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return false, pi, err
	}
	req.Header.Set("Accept", "application/json")
	e.setAuthHeader(req)

	resp, err := e.opts.client.Do(req)
	if err != nil {
		return true, pi, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		rerr = err
		isErrorRetryable = true
		return
	}

	rateLimited := func() (isErrorRetryable bool, pageInfo PageInfo, rerr error) {

		waitTime := time.Minute * 3

		sdk.LogWarn(e.opts.Logger, "api request failed due to throttling, the quota of 600 calls has been reached, will sleep for 3m and retry", "retryThrottled", retryThrottled)

		paused := time.Now()
		resumeDate := paused.Add(waitTime)
		sdk.LogWarn(e.opts.Logger, "gitlab paused, it will resume in %v, resume data %v", waitTime, resumeDate)

		time.Sleep(waitTime)

		sdk.LogWarn(e.opts.Logger, fmt.Sprintf("gitlab resumed, time elapsed %v", time.Since(paused)))

		return true, PageInfo{}, fmt.Errorf("too many requests")

	}

	if resp.StatusCode != http.StatusOK {

		if resp.StatusCode == http.StatusTooManyRequests {
			return rateLimited()
		}

		if resp.StatusCode == http.StatusForbidden {

			var errorR *errorResponse

			er := json.Unmarshal([]byte(b), &errorR)
			if er != nil {
				return false, pi, fmt.Errorf("unmarshal error %s", er)
			}

			return false, pi, fmt.Errorf("%s, %s, scopes required: api, read_user, read_repository", errorR.Error, errorR.ErrorDescription)
		}

		sdk.LogWarn(e.opts.Logger, "gitlab returned invalid status code, retrying", "code", resp.StatusCode, "retry", retryThrottled, "url", req.URL.String())

		return true, pi, fmt.Errorf("request with status %d", resp.StatusCode)
	}
	err = json.Unmarshal(b, &r.Response)
	if err != nil {
		rerr = err
		return
	}

	rawPageSize := resp.Header.Get("X-Per-Page")

	var pageSize int
	if rawPageSize != "" {
		pageSize, err = strconv.Atoi(rawPageSize)
		if err != nil {
			return false, pi, err
		}
	}

	rawTotalSize := resp.Header.Get("X-Total")

	var total int
	if rawTotalSize != "" {
		total, err = strconv.Atoi(rawTotalSize)
		if err != nil {
			return false, pi, err
		}
	}

	return false, PageInfo{
		PageSize:   pageSize,
		NextPage:   resp.Header.Get("X-Next-Page"),
		Page:       resp.Header.Get("X-Page"),
		TotalPages: resp.Header.Get("X-Total-Pages"),
		Total:      total,
	}, nil
}
