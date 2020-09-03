package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/pinpt/agent.next/sdk"
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
			MergedAt:       at,
			SourceBranch:   "master",
			Draft:          true,
			State:          "closed",
			MergeCommitSHA: "",
		},
		WebURL:    "https://api.gitlab.com/",
		CreatedAt: at,
		UpdatedAt: at,
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
			Full string `json:"full"`
		}{
			Full: "pinpt/test!2",
		},
	}

	sourceCodePR := pr.toSourceCodePullRequest(logger, customerID, repoID, refType)

	sourceCodePR.ToMap()

	assert.Equal(customerID, sourceCodePR.CustomerID)
	assert.Equal(repoID, sourceCodePR.RepoID)
	assert.Equal(refType, sourceCodePR.RefType)
	assert.Equal(pr.References.Full, sourceCodePR.Identifier)
	assert.Equal(pr.Title, sourceCodePR.Title)
	assert.Equal(setHTMLPRDescription(pr.Description), sourceCodePR.Description)
	assert.Equal(sdk.TimeToEpoch(pr.CreatedAt), sourceCodePR.CreatedDate.Epoch)
	assert.Equal(sdk.TimeToEpoch(pr.UpdatedAt), sourceCodePR.UpdatedDate.Epoch)
	assert.Equal(int64(0), sourceCodePR.MergedDate.Epoch)
	assert.Equal(pr.SourceBranch, sourceCodePR.BranchName)
	assert.Equal(pr.MergeCommitSHA, sourceCodePR.MergeSha)
	assert.Equal(pr.WebURL, sourceCodePR.URL)
	assert.Equal(strconv.FormatInt(pr.Author.ID, 10), sourceCodePR.CreatedByRefID)
	assert.Equal(strconv.FormatInt(pr.ClosedBy.ID, 10), sourceCodePR.ClosedByRefID)
	assert.Equal("", sourceCodePR.MergedByRefID)

	sourceCodePR.ToMap()

	assert.Equal(sdk.NewSourceCodePullRequestID(customerID, strconv.FormatInt(pr.ID, 10), refType, repoID), sourceCodePR.ID)

}
func TestWebHookPRToSrcCode(t *testing.T) {

	assert := assert.New(t)
	logger := sdk.NewNoOpTestLogger()
	customerID := "123"
	repoID := "w45fdc4"
	refType := "gitlab"

	createdAtDate := "2020-09-02 20:01:42 UTC"
	updatedAtDate := "2020-09-02 23:12:40 UTC"

	createdAt, err := time.Parse("2006-01-02 15:04:05 MST", createdAtDate)
	assert.NoError(err)

	updatedAt, err := time.Parse("2006-01-02 15:04:05 MST", updatedAtDate)
	assert.NoError(err)

	// whDateFormat := at.Format("2006-01-02 15:04:05 MST")

	body := `{
        "assignee_id": null,
        "author_id": 2,
        "created_at": "` + createdAtDate + `",
        "description": "",
        "head_pipeline_id": null,
        "id": 6,
        "iid": 4,
        "last_edited_at": null,
        "last_edited_by_id": null,
        "merge_commit_sha": null,
        "merge_error": null,
        "merge_params": {
            "force_remove_source_branch": "1"
        },
        "merge_status": "can_be_merged",
        "merge_user_id": null,
        "merge_when_pipeline_succeeds": false,
        "milestone_id": null,
        "source_branch": "mr3",
        "source_project_id": 3,
        "state_id": 1,
        "target_branch": "master",
        "target_project_id": 3,
        "time_estimate": 0,
        "title": "test",
        "updated_at": "` + updatedAtDate + `",
        "updated_by_id": null,
        "url": "http://gitlab.example.com/localgroup1/apilocal/-/merge_requests/4",
        "source": {
            "id": 3,
            "name": "apilocal",
            "description": "test api",
            "web_url": "http://gitlab.example.com/localgroup1/apilocal",
            "avatar_url": null,
            "git_ssh_url": "g**@git***.example.com:localgroup1/apilocal.git",
            "git_http_url": "http://gitlab.example.com/localgroup1/apilocal.git",
            "namespace": "localgroup1",
            "visibility_level": 0,
            "path_with_namespace": "localgroup1/apilocal",
            "default_branch": "master",
            "ci_config_path": null,
            "homepage": "http://gitlab.example.com/localgroup1/apilocal",
            "url": "g**@git***.example.com:localgroup1/apilocal.git",
            "ssh_url": "g**@git***.example.com:localgroup1/apilocal.git",
            "http_url": "http://gitlab.example.com/localgroup1/apilocal.git"
        },
        "target": {
            "id": 3,
            "name": "apilocal",
            "description": "test api",
            "web_url": "http://gitlab.example.com/localgroup1/apilocal",
            "avatar_url": null,
            "git_ssh_url": "g**@git***.example.com:localgroup1/apilocal.git",
            "git_http_url": "http://gitlab.example.com/localgroup1/apilocal.git",
            "namespace": "localgroup1",
            "visibility_level": 0,
            "path_with_namespace": "localgroup1/apilocal",
            "default_branch": "master",
            "ci_config_path": null,
            "homepage": "http://gitlab.example.com/localgroup1/apilocal",
            "url": "g**@git***.example.com:localgroup1/apilocal.git",
            "ssh_url": "g**@git***.example.com:localgroup1/apilocal.git",
            "http_url": "http://gitlab.example.com/localgroup1/apilocal.git"
        },
        "last_commit": {
            "id": "7a7f609819718f146f7cbb35311dfbd2340412fb",
            "message": "test\\n",
            "title": "test",
            "timestamp": "2020-09-02T15:00:16-05:00",
            "url": "http://gitlab.example.com/localgroup1/apilocal/-/commit/7a7f609819718f146f7cbb35311dfbd2340412fb",
            "author": {
                "name": "J Cas Orz",
                "email": "cor***@pinp****.com"
            }
        },
        "work_in_progress": false,
        "total_time_spent": 0,
        "human_total_time_spent": null,
        "human_time_estimate": null,
        "assignee_ids": [],
        "state": "opened",
        "action": "reopen"
    }`

	pr := &WebhookPullRequest{
		// CommonPullRequestFields: &CommonPullRequestFields{
		// 	ID:             1,
		// 	IID:            2,
		// 	Title:          "API MR1",
		// 	Description:    "Description Merge Request",
		// 	MergedAt:       at,
		// 	SourceBranch:   "master",
		// 	Draft:          true,
		// 	State:          "merged",
		// 	MergeCommitSHA: "b160ae890592d17bc17335e2",
		// },
		// CreatedAt:  whDateFormat,
		// UpdatedAt:  whDateFormat,
		// URL:        "https://api.gitlab.com/",
		// AuthorID:   1,
		// ClosedByID: 2,
		// MergedByID: 3,
	}

	err = json.Unmarshal([]byte(body), &pr)
	assert.NoError(err)

	sourceCodePR, err := pr.ToSourceCodePullRequest(logger, customerID, repoID, refType)
	assert.NoError(err)

	assert.Equal(customerID, sourceCodePR.CustomerID)
	assert.Equal(repoID, sourceCodePR.RepoID)
	assert.Equal(refType, sourceCodePR.RefType)
	assert.Equal(fmt.Sprintf("!%d", pr.IID), sourceCodePR.Identifier)
	assert.Equal(pr.Title, sourceCodePR.Title)
	assert.Equal(setHTMLPRDescription(pr.Description), sourceCodePR.Description)
	assert.Equal(sdk.TimeToEpoch(createdAt.Round(time.Second)), sourceCodePR.CreatedDate.Epoch)
	assert.Equal(sdk.TimeToEpoch(updatedAt.Round(time.Second)), sourceCodePR.UpdatedDate.Epoch)
	assert.Equal(sdk.TimeToEpoch(pr.MergedAt), sourceCodePR.MergedDate.Epoch)
	assert.Equal(pr.SourceBranch, sourceCodePR.BranchName)
	assert.Equal(pr.MergeCommitSHA, sourceCodePR.MergeSha)
	assert.Equal(pr.URL, sourceCodePR.URL)
	assert.Equal(strconv.FormatInt(pr.AuthorID, 10), sourceCodePR.CreatedByRefID)
	assert.Equal("", sourceCodePR.ClosedByRefID)
	assert.Equal("", sourceCodePR.MergedByRefID)

	sourceCodePR.ToMap()

	assert.Equal(sdk.NewSourceCodePullRequestID(customerID, strconv.FormatInt(pr.ID, 10), refType, repoID), sourceCodePR.ID)

}
func TestWebHookPRToSrcCode2(t *testing.T) {

	assert := assert.New(t)
	logger := sdk.NewNoOpTestLogger()
	customerID := "123"
	repoID := "w45fdc4"
	refType := "gitlab"

	createdAtDate := "2020-09-02 20:01:42 UTC"
	updatedAtDate := "2020-09-02 23:12:40 UTC"

	createdAt, err := time.Parse("2006-01-02 15:04:05 MST", createdAtDate)
	assert.NoError(err)

	updatedAt, err := time.Parse("2006-01-02 15:04:05 MST", updatedAtDate)
	assert.NoError(err)

	// whDateFormat := at.Format("2006-01-02 15:04:05 MST")

	body := `{
        "assignee_id": null,
        "author_id": 2,
        "created_at": "` + createdAtDate + `",
        "description": "",
        "head_pipeline_id": null,
        "id": 6,
        "iid": 4,
        "last_edited_at": null,
        "last_edited_by_id": null,
        "merge_commit_sha": null,
        "merge_error": null,
        "merge_params": {
            "force_remove_source_branch": "1"
        },
        "merge_status": "can_be_merged",
        "merge_user_id": null,
        "merge_when_pipeline_succeeds": false,
        "milestone_id": null,
        "source_branch": "mr3",
        "source_project_id": 3,
        "state_id": 2,
        "target_branch": "master",
        "target_project_id": 3,
        "time_estimate": 0,
        "title": "test",
        "updated_at": "` + updatedAtDate + `",
        "updated_by_id": null,
        "url": "http://gitlab.example.com/localgroup1/apilocal/-/merge_requests/4",
        "source": {
            "id": 3,
            "name": "apilocal",
            "description": "test api",
            "web_url": "http://gitlab.example.com/localgroup1/apilocal",
            "avatar_url": null,
            "git_ssh_url": "g**@git***.example.com:localgroup1/apilocal.git",
            "git_http_url": "http://gitlab.example.com/localgroup1/apilocal.git",
            "namespace": "localgroup1",
            "visibility_level": 0,
            "path_with_namespace": "localgroup1/apilocal",
            "default_branch": "master",
            "ci_config_path": null,
            "homepage": "http://gitlab.example.com/localgroup1/apilocal",
            "url": "g**@git***.example.com:localgroup1/apilocal.git",
            "ssh_url": "g**@git***.example.com:localgroup1/apilocal.git",
            "http_url": "http://gitlab.example.com/localgroup1/apilocal.git"
        },
        "target": {
            "id": 3,
            "name": "apilocal",
            "description": "test api",
            "web_url": "http://gitlab.example.com/localgroup1/apilocal",
            "avatar_url": null,
            "git_ssh_url": "g**@git***.example.com:localgroup1/apilocal.git",
            "git_http_url": "http://gitlab.example.com/localgroup1/apilocal.git",
            "namespace": "localgroup1",
            "visibility_level": 0,
            "path_with_namespace": "localgroup1/apilocal",
            "default_branch": "master",
            "ci_config_path": null,
            "homepage": "http://gitlab.example.com/localgroup1/apilocal",
            "url": "g**@git***.example.com:localgroup1/apilocal.git",
            "ssh_url": "g**@git***.example.com:localgroup1/apilocal.git",
            "http_url": "http://gitlab.example.com/localgroup1/apilocal.git"
        },
        "last_commit": {
            "id": "7a7f609819718f146f7cbb35311dfbd2340412fb",
            "message": "test\\n",
            "title": "test",
            "timestamp": "2020-09-02T15:00:16-05:00",
            "url": "http://gitlab.example.com/localgroup1/apilocal/-/commit/7a7f609819718f146f7cbb35311dfbd2340412fb",
            "author": {
                "name": "J Ca Orz",
                "email": "cor***@pinp****.com"
            }
        },
        "work_in_progress": false,
        "total_time_spent": 0,
        "human_total_time_spent": null,
        "human_time_estimate": null,
        "assignee_ids": [],
        "state": "closed",
        "action": "close"
    }`

	pr := &WebhookPullRequest{}

	err = json.Unmarshal([]byte(body), &pr)
	assert.NoError(err)

	sourceCodePR, err := pr.ToSourceCodePullRequest(logger, customerID, repoID, refType)
	assert.NoError(err)

	assert.Equal(customerID, sourceCodePR.CustomerID)
	assert.Equal(repoID, sourceCodePR.RepoID)
	assert.Equal(refType, sourceCodePR.RefType)
	assert.Equal(fmt.Sprintf("!%d", pr.IID), sourceCodePR.Identifier)
	assert.Equal(pr.Title, sourceCodePR.Title)
	assert.Equal(setHTMLPRDescription(pr.Description), sourceCodePR.Description)
	assert.Equal(sdk.TimeToEpoch(createdAt.Round(time.Second)), sourceCodePR.CreatedDate.Epoch)
	assert.Equal(sdk.TimeToEpoch(updatedAt.Round(time.Second)), sourceCodePR.UpdatedDate.Epoch)
	assert.Equal(sdk.TimeToEpoch(pr.MergedAt), sourceCodePR.MergedDate.Epoch)
	assert.Equal(pr.SourceBranch, sourceCodePR.BranchName)
	assert.Equal(pr.MergeCommitSHA, sourceCodePR.MergeSha)
	assert.Equal(pr.URL, sourceCodePR.URL)
	assert.Equal(strconv.FormatInt(pr.AuthorID, 10), sourceCodePR.CreatedByRefID)
	assert.Equal("", sourceCodePR.ClosedByRefID)
	assert.Equal("", sourceCodePR.MergedByRefID)

	sourceCodePR.ToMap()

	assert.Equal(sdk.NewSourceCodePullRequestID(customerID, strconv.FormatInt(pr.ID, 10), refType, repoID), sourceCodePR.ID)

}
