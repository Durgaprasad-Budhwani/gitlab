package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidTier(t *testing.T) {

	assert := assert.New(t)

	type namespace struct {
		//json.RawMessage
		MembersCountWithDescendants *int `json:"members_count_with_descendants"`
	}

	body := `{"members_count_with_descendants":1}`

	var g namespace

	err := json.Unmarshal([]byte(body), &g)
	assert.NoError(err)

	t.Log("value1", *g.MembersCountWithDescendants)

	//assert.True(isValidTier(g.RawMessage))

	var g2 namespace

	body = "{}"

	err = json.Unmarshal([]byte(body), &g2)
	assert.NoError(err)

	t.Log("value2", *g2.MembersCountWithDescendants)

	//assert.False(isValidTier(g.RawMessage))
}


func TestNamespacesPage(t *testing.T){

	//BaseURL string
	//Logger  sdk.Logger
	//Get     func(url string, params url.Values, response interface{}) (NextPage, error)
	//Post    func(url string, params url.Values, data io.Reader, response interface{}) (NextPage, error)
	//Delete  func(url string, params url.Values, response interface{}) (NextPage, error)
	//Put     func(url string, params url.Values, data io.Reader, response interface{}) (NextPage, error)
	//
	//CustomerID string
	//RefType    string
	//
	//UserEmailMap          map[string]string
	//IntegrationInstanceID string
	//Pipe                  sdk.Pipe
	//UserManager           UserManager2
	//WorkManager           WorkManagerI
	//State                 sdk.State
	//Historical            bool
	//GraphRequester        GraphqlRequester

	//config := sdk.Config{}
	//
	//logger := sdk.NewNoOpTestLogger()

	//apiURL, client, graphql, err := NewHTTPClient(logger, config, manager)
	//if err != nil {
	//	rerr = err
	//	return
	//}
	//
	//qc := QueryContext{}
	//r := NewRequester(logger, client, 10)
	//qc.Get = r.Get
	//
	//assert := assert.New(t)
	//
	//NamespacesPage()

}