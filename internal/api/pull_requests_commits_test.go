package api

import (
	"testing"
	"time"

	"github.com/pinpt/agent/sdk"
	"github.com/stretchr/testify/assert"
)

func TestApiPRCommitToSrcCode(t *testing.T) {

	assert := assert.New(t)

	customerID := "123"
	repoID := "w45fdc4"
	pullRequestID := "43287sd"
	refType := "gitlab"

	at, err := time.Parse(time.RFC3339, "2020-07-20T17:28:08.571Z")
	assert.NoError(err)

	prc := &PrCommit{
		CommonCommitFields: &CommonCommitFields{
			ID:      "6aecf4246f0c6da6d5c3024090ad2a0b0a682e4c",
			Message: "test",
		},
		WebURL:         "https://api.gitlab.com/",
		AuthorEmail:    "t876q2345ruwea",
		CommitterEmail: "q2827weduiysd",
		CreatedAt:      at,
	}

	scprc := prc.ToSourceCodePullRequestCommit(customerID, refType, repoID, pullRequestID)

	assert.Equal(customerID, scprc.CustomerID)
	assert.Equal(refType, scprc.RefType)
	assert.Equal(repoID, scprc.RepoID)
	assert.Equal(pullRequestID, scprc.PullRequestID)
	assert.Equal(prc.WebURL, scprc.URL)
	assert.Equal(CodeCommitEmail(customerID, prc.AuthorEmail), scprc.AuthorRefID)
	assert.Equal(CodeCommitEmail(customerID, prc.CommitterEmail), scprc.CommitterRefID)

	assert.Equal(prc.CreatedAt, prc.CreatedAt)

	scprc.ToMap()

	assert.Equal(sdk.NewSourceCodePullRequestCommentID(customerID, prc.ID, refType, repoID), scprc.ID)

}

func TestWebHookPRCommitToSrcCode(t *testing.T) {

	assert := assert.New(t)

	customerID := "123"
	repoID := "w45fdc4"
	pullRequestID := "43287sd"
	refType := "gitlab"

	at, err := time.Parse(time.RFC3339, "2020-07-20T17:28:08.571Z")
	assert.NoError(err)

	whprc := &WhCommit{
		CommonCommitFields: &CommonCommitFields{
			ID:      "6aecf4246f0c6da6d5c3024090ad2a0b0a682e4c",
			Message: "test",
		},
		URL: "https://api.gitlab.com/",
		Author: struct {
			Email string `json:"email"`
		}{
			Email: "myemail@gmail.com",
		},
		Timestamp: at,
	}

	scprc := whprc.ToSourceCodePullRequestCommit(customerID, refType, repoID, pullRequestID)

	assert.Equal(customerID, scprc.CustomerID)
	assert.Equal(refType, scprc.RefType)
	assert.Equal(repoID, scprc.RepoID)
	assert.Equal(pullRequestID, scprc.PullRequestID)
	assert.Equal(whprc.URL, scprc.URL)
	assert.Equal(CodeCommitEmail(customerID, whprc.Author.Email), scprc.AuthorRefID)

	assert.Equal(sdk.TimeToEpoch(whprc.Timestamp), scprc.CreatedDate.Epoch)

	scprc.ToMap()

	assert.Equal(sdk.NewSourceCodePullRequestCommentID(customerID, whprc.ID, refType, repoID), scprc.ID)

}
