package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/pinpt/go-common/v10/log"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
	assert2 "github.com/stretchr/testify/assert"
)

type mapValue struct {
	expire *time.Time
	value  interface{}
}

type testState struct {
	sync.Map
}

func (t *testState) Set(key string, value interface{}) error {
	t.Store(key, mapValue{
		expire: nil,
		value:  value,
	})
	return nil
}
func (t *testState) SetWithExpires(key string, value interface{}, expiry time.Duration) error {
	ex := time.Now().Add(expiry)
	t.Store(key, mapValue{
		expire: &ex,
		value:  value,
	})
	return nil
}
func (t *testState) Get(key string, out interface{}) (bool, error) {

	value, ok := t.Load(key)
	if !ok {
		return false, nil
	}

	if value.(mapValue).expire != nil {
		if time.Now().After(*value.(mapValue).expire) {
			return false, nil
		}
	}

	reflect.ValueOf(out).Elem().Set(reflect.ValueOf(value.(mapValue).value))

	return true, nil
}
func (t *testState) Exists(key string) bool {
	_, ok := t.Map.Load(key)
	return ok
}
func (t *testState) Delete(key string) error {
	t.Map.Delete(key)
	return nil
}
func (t *testState) Flush() error {
	return nil
}

var _ sdk.State = (*testState)(nil)

func TestState(t *testing.T) {
	state := testState{}
	state.Set("key", "myvalue")

	var response string
	state.Get("key", &response)
	fmt.Println("response", response)
}

type testPipe struct {
	// Written is all the models written to this pipe in order
	Written []sdk.Model
	// Closed is set to true every time Close() is called
	Closed bool
	// Flushed is set to true every time Flush() is called
	Flushed bool

	// WriteErr is returned by Write
	WriteErr error
	// FlushErr is returned by Flush
	FlushErr error
	// CloseErr is returned by Close
	CloseErr error
}

var _ sdk.Pipe = (*testPipe)(nil)

// Write a model back to the output system
func (p testPipe) Write(object sdk.Model) error {
	p.Written = append(p.Written, object)
	return p.WriteErr
}

// Flush will tell the pipe to flush any pending data
func (p testPipe) Flush() error {
	p.Flushed = true
	return p.FlushErr
}

// Close is called when the integration has completed and no more data will be sent
func (p testPipe) Close() error {
	p.Closed = true
	return p.CloseErr
}

func (p testPipe) printOUTContent() {
	for _, l := range p.Written {
		fmt.Println("l", l)
	}
}

func TestGetAllNamespaces(t *testing.T) {

	assert := assert2.New(t)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	ns := []*api.GitlabNamespace{
		{
			ID:                          347835,
			Name:                        "J Carlos O",
			Path:                        "josecordaz",
			FullPath:                    "josecordaz",
			MembersCountWithDescendants: nil,
			AvatarURL:                   "https://secure.gravatar.com/avatar/dbaea9b",
			ParentID:                    0,
			Kind:                        "user",
		},
	}

	httpmock.RegisterResponder("GET", "https://gitlab.com/api/v4/namespaces",
		func(req *http.Request) (*http.Response, error) {
			response, err := httpmock.NewJsonResponse(200, ns)
			if err != nil {
				return nil, err
			}
			//response.Header.Set("","")
			return response, nil
		})

	logger := log.NewLogger(os.Stdout, log.JSONLogFormat, log.DarkLogColorTheme, log.DebugLevel, "test")
	//logger := sdk.NewNoOpTestLogger()

	pipe := testPipe{}
	state := &testState{}

	gURL := "https://gitlab.com/api/v4"

	u, err := url.Parse(gURL)
	assert.NoError(err)

	qc := &api.QueryContext2{
		URL: u,
		Get: func(logger sdk.Logger, path string, params url.Values, response interface{}) (api.NextPage, error) {
			assert.Equal("1", params.Get("page"))
			assert.Equal("100", params.Get("per_page"))
			assert.Equal("true", params.Get("top_level_only"))

			assert.Empty(params.Get("order_by"))
			assert.Equal("namespaces", path)

			req, err := http.NewRequest(http.MethodGet, gURL+"/"+path, nil)
			assert.NoError(err)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(err)

			err = json.NewDecoder(resp.Body).Decode(response)
			assert.NoError(err)

			return api.NextPage(""), nil
		},
	}

	config := sdk.NewConfig(map[string]interface{}{})

	ge, err := newGitlabExport(
		logger,
		qc,
		sdk.StringPointer("1234"),
		func(logger sdk.Logger, namespaceID string, name string, isArchived bool) bool {
			return false
		},
		"4321",
		true,
		pipe,
		state,
		config,
	)
	assert.NoError(err)

	namespaces := make(chan *Namespace)
	err = ge.getSelectedNamespacesIfAny(logger, namespaces)
	assert.NoError(err)
	assert.NotEmpty(namespaces)

	for n := range namespaces {
		assert.Equal(toNamespace(ns[0]), n)
	}

	pipe.printOUTContent()

}
