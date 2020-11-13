package api

import (
	"encoding/json"
	"testing"

	ast "github.com/stretchr/testify/assert"
)

func TestIsValidTier(t *testing.T) {

	assert := ast.New(t)

	type namespace struct {
		json.RawMessage
		MembersCountWithDescendants string `json:"members_count_with_descendants"`
	}

	body := `{"members_count_with_descendants":1}`

	var g namespace

	err := json.Unmarshal([]byte(body), &g)
	assert.NoError(err)

	assert.True(isValidTier(g.RawMessage))

	body = "{}"

	err = json.Unmarshal([]byte(body), &g)
	assert.NoError(err)

	assert.False(isValidTier(g.RawMessage))
}
