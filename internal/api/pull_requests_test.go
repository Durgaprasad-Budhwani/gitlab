package api

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/datetime"
	"github.com/stretchr/testify/assert"
)

func TestApiPRToSrcCode(t *testing.T) {

	assert := assert.New(t)

	logger := sdk.NewNoOpTestLogger()
	customerID := "123"
	repoID := "w45fdc4"
	refType := "gitlab"

	at, err := time.Parse(time.RFC3339, "2020-07-20T17:28:08.571Z")
	assert.NoError(err)

	pr := &apiPullRequest{
		CommonPullRequestFields: &CommonPullRequestFields{
			ID:             1,
			IID:            2,
			Title:          "API MR1",
			Description:    "Description Merge Request",
			CreatedAt:      at,
			UpdatedAt:      at,
			MergedAt:       at,
			SourceBranch:   "master",
			Draft:          true,
			State:          "merged",
			MergeCommitSHA: "b160ae890592d17bc17335e2",
		},
		WebURL: "https://api.gitlab.com/",
		Author: author{
			ID: 1,
		},
		ClosedBy: author{
			ID: 2,
		},
		MergedBy: author{
			ID: 3,
		},
		References: struct {
			Short string `json:"short"`
		}{
			Short: "!2",
		},
	}

	sourceCodePR := pr.toSourceCodePullRequest(logger, customerID, repoID, refType)

	assert.Equal(customerID, sourceCodePR.CustomerID)
	assert.Equal(repoID, sourceCodePR.RepoID)
	assert.Equal(refType, sourceCodePR.RefType)
	assert.Equal(fmt.Sprintf("!%d", pr.IID), sourceCodePR.Identifier)
	assert.Equal(pr.Title, sourceCodePR.Title)
	assert.Equal(setHTMLPRDescription(pr.Description), sourceCodePR.Description)
	assert.Equal(datetime.TimeToEpoch(pr.CreatedAt), sourceCodePR.CreatedDate.Epoch)
	assert.Equal(datetime.TimeToEpoch(pr.UpdatedAt), sourceCodePR.UpdatedDate.Epoch)
	assert.Equal(datetime.TimeToEpoch(pr.MergedAt), sourceCodePR.MergedDate.Epoch)
	assert.Equal(pr.SourceBranch, sourceCodePR.BranchName)
	assert.Equal(pr.MergeCommitSHA, sourceCodePR.MergeSha)
	assert.Equal(pr.WebURL, sourceCodePR.URL)
	assert.Equal(strconv.FormatInt(pr.Author.ID, 10), sourceCodePR.CreatedByRefID)
	assert.Equal(strconv.FormatInt(pr.ClosedBy.ID, 10), sourceCodePR.ClosedByRefID)
	assert.Equal(strconv.FormatInt(pr.MergedBy.ID, 10), sourceCodePR.MergedByRefID)
	assert.Equal(sdk.NewSourceCodePullRequestID(customerID, strconv.FormatInt(pr.ID, 10), refType, repoID), sourceCodePR.ID)

}
func TestWebHookPRToSrcCode(t *testing.T) {

	assert := assert.New(t)
	logger := sdk.NewNoOpTestLogger()
	customerID := "123"
	repoID := "w45fdc4"
	refType := "gitlab"

	at, err := time.Parse(time.RFC3339, "2020-07-20T17:28:08.571Z")
	assert.NoError(err)

	pr := &WebhookPullRequest{
		CommonPullRequestFields: &CommonPullRequestFields{
			ID:             1,
			IID:            2,
			Title:          "API MR1",
			Description:    "Description Merge Request",
			CreatedAt:      at,
			UpdatedAt:      at,
			MergedAt:       at,
			SourceBranch:   "master",
			Draft:          true,
			State:          "merged",
			MergeCommitSHA: "b160ae890592d17bc17335e2",
		},
		URL:        "https://api.gitlab.com/",
		AuthorID:   1,
		ClosedByID: 2,
		MergedByID: 3,
	}

	sourceCodePR := pr.ToSourceCodePullRequest(logger, customerID, repoID, refType)

	assert.Equal(customerID, sourceCodePR.CustomerID)
	assert.Equal(repoID, sourceCodePR.RepoID)
	assert.Equal(refType, sourceCodePR.RefType)
	assert.Equal(fmt.Sprintf("!%d", pr.IID), sourceCodePR.Identifier)
	assert.Equal(pr.Title, sourceCodePR.Title)
	assert.Equal(setHTMLPRDescription(pr.Description), sourceCodePR.Description)
	assert.Equal(datetime.TimeToEpoch(pr.CreatedAt), sourceCodePR.CreatedDate.Epoch)
	assert.Equal(datetime.TimeToEpoch(pr.UpdatedAt), sourceCodePR.UpdatedDate.Epoch)
	assert.Equal(datetime.TimeToEpoch(pr.MergedAt), sourceCodePR.MergedDate.Epoch)
	assert.Equal(pr.SourceBranch, sourceCodePR.BranchName)
	assert.Equal(pr.MergeCommitSHA, sourceCodePR.MergeSha)
	assert.Equal(pr.URL, sourceCodePR.URL)
	assert.Equal(strconv.FormatInt(pr.AuthorID, 10), sourceCodePR.CreatedByRefID)
	assert.Equal(strconv.FormatInt(pr.ClosedByID, 10), sourceCodePR.ClosedByRefID)
	assert.Equal(strconv.FormatInt(pr.MergedByID, 10), sourceCodePR.MergedByRefID)
	assert.Equal(sdk.NewSourceCodePullRequestID(customerID, strconv.FormatInt(pr.ID, 10), refType, repoID), sourceCodePR.ID)

}
