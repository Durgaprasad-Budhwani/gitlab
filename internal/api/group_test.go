package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestisValidTier(t *testing.T) {

	assert := assert.New(t)

	type group struct {
		json.RawMessage
		MarkedForDeletring string `json:"marked_for_deletion"`
	}

	body := `{"marked_for_deletion":true}`

	var g group

	err := json.Unmarshal([]byte(body), &g)
	assert.NoError(err)

	assert.True(isValidTier(g.RawMessage))

	body = "{}"

	err = json.Unmarshal([]byte(body), &g)
	assert.NoError(err)

	assert.False(isValidTier(g.RawMessage))
}
