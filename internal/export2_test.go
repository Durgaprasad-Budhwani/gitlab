package internal

import (
	"encoding/json"
	"github.com/jarcoal/httpmock"
	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
	"github.com/pinpt/go-common/v10/log"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestExport(t *testing.T){

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
		{
			ID:                          376242,
			Name:                        "not selected",
			Path:                        "notselected",
			FullPath:                    "notselected",
			MembersCountWithDescendants: nil,
			AvatarURL:                   "https://secure.gravatar.com/avatar/ns",
			ParentID:                    0,
			Kind:                        "user",
		},
	}

	createdAt, _ := time.Parse(time.RFC3339,"2020-08-27T18:39:35.015Z")

	repos := []api.GitlabProject{
		{
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
			RefID: 23454,
			FullName: "josecordaz/test",
			Description: "test project for unit test",
			WebURL: "https://gitlab.com/josecordaz/test",
			Archived: false,
			DefaultBranch: "master",
			Visibility: "",
			ForkedFromProject: nil,
			Owner: struct {
				RefID int64 `json:"id"`
			}{
				RefID: 2345,
			},
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

	httpmock.RegisterResponder("GET", "https://gitlab.com/api/v4/groups/347835/projects",
		func(req *http.Request) (*http.Response, error) {
			response, err := httpmock.NewJsonResponse(200, repos)
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

	accounts := sdk.ConfigAccounts{
		"347835": &sdk.ConfigAccount{
			ID: "347835",
			Selected: sdk.BoolPointer(true),
			Type: sdk.ConfigAccountTypeUser,
		},
		"376242": &sdk.ConfigAccount{
			ID: "376242",
			Selected: sdk.BoolPointer(false),
			Type: sdk.ConfigAccountTypeUser,
		},
	}
	bts, err := json.Marshal(accounts)
	assert.NoError(err)

	config := sdk.NewConfig(map[string]interface{}{
		"accounts": string(bts),
	})

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

	ge, err := newGitlabExport(logger, qc,
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

	err = ge.Export(logger)
	assert.NoError(err)

	// assert.Equal(true,false)

	pipe.printOUTContent()
}