package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/datetime"
)

type PullRequest struct {
	*sdk.SourceCodePullRequest
	IID           string
	LastCommitSHA string
}

type CommonPullRequestFields struct {
	ID             int64     `json:"id"`
	IID            int64     `json:"iid"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
	ClosedAt       time.Time `json:"closed_at"`
	MergedAt       time.Time `json:"merged_at"`
	SourceBranch   string    `json:"source_branch"`
	Draft          bool      `json:"work_in_progress"`
	State          string    `json:"state"`
	MergeCommitSHA string    `json:"merge_commit_sha"`
}

func (c *CommonPullRequestFields) commonToSourceCodePullRequest(logger sdk.Logger, customerID, repoID string, refType string) (pr *sdk.SourceCodePullRequest) {

	pr = &sdk.SourceCodePullRequest{}
	prRefID := strconv.FormatInt(c.ID, 10)
	pr.ID = sdk.NewSourceCodePullRequestID(customerID, prRefID, refType, repoID)
	pr.CustomerID = customerID
	pr.RefType = refType
	pr.RefID = prRefID
	pr.RepoID = repoID

	pr.Title = c.Title
	pr.BranchName = c.SourceBranch
	pr.Description = setHTMLPRDescription(c.Description)
	datetime.ConvertToModel(c.CreatedAt, &pr.CreatedDate)
	datetime.ConvertToModel(c.MergedAt, &pr.MergedDate)
	datetime.ConvertToModel(c.ClosedAt, &pr.ClosedDate)
	datetime.ConvertToModel(c.UpdatedAt, &pr.UpdatedDate)
	pr.Draft = c.Draft
	switch c.State {
	case "opened":
		pr.Status = sdk.SourceCodePullRequestStatusOpen
	case "closed":
		pr.Status = sdk.SourceCodePullRequestStatusClosed
	case "locked":
		pr.Status = sdk.SourceCodePullRequestStatusLocked
	case "merged":
		pr.MergeSha = c.MergeCommitSHA
		pr.MergeCommitID = sdk.NewSourceCodePullRequestCommentID(customerID, c.MergeCommitSHA, refType, repoID)
		pr.Status = sdk.SourceCodePullRequestStatusMerged
	default:
		sdk.LogError(logger, "PR has an unknown state", "state", c.State, "ref_id", pr.RefID)
	}

	return
}

func setHTMLPRDescription(description string) string {
	return `<div class="source-gitlab">` + sdk.ConvertMarkdownToHTML(description) + "</div>"
}

type author struct {
	ID int64 `json:"id"`
}

type apiPullRequest struct {
	*CommonPullRequestFields
	WebURL     string `json:"web_url"`
	Author     author `json:"author"`
	ClosedBy   author `json:"closed_by"`
	MergedBy   author `json:"merged_by"`
	References struct {
		Short string `json:"short"` // this looks how we display in Gitlab such as !1
	} `json:"references"`
}

func (amr *apiPullRequest) toSourceCodePullRequest(logger sdk.Logger, customerID, repoID string, refType string) (pr *sdk.SourceCodePullRequest) {

	pr = amr.CommonPullRequestFields.commonToSourceCodePullRequest(logger, customerID, repoID, refType)

	pr.URL = amr.WebURL
	pr.CreatedByRefID = strconv.FormatInt(amr.Author.ID, 10)
	pr.ClosedByRefID = strconv.FormatInt(amr.ClosedBy.ID, 10)
	pr.MergedByRefID = strconv.FormatInt(amr.MergedBy.ID, 10)
	pr.Identifier = amr.References.Short

	return
}

// WebhookPullRequest
type WebhookPullRequest struct {
	*CommonPullRequestFields
	URL        string `json:"url"`
	AuthorID   int64  `json:"author_id"`
	ClosedByID int64  `json:"closed_by_id"`
	MergedByID int64  `json:"merged_by_id"`
	Action     string `json:"action"`
}

func (wpr *WebhookPullRequest) ToSourceCodePullRequest(logger sdk.Logger, customerID, repoID string, refType string) (pr *sdk.SourceCodePullRequest) {

	pr = wpr.CommonPullRequestFields.commonToSourceCodePullRequest(logger, customerID, repoID, refType)

	pr.URL = wpr.URL
	pr.CreatedByRefID = strconv.FormatInt(wpr.AuthorID, 10)
	pr.ClosedByRefID = strconv.FormatInt(wpr.ClosedByID, 10)
	pr.MergedByRefID = strconv.FormatInt(wpr.MergedByID, 10)
	pr.Identifier = fmt.Sprintf("!%d", wpr.IID)

	return
}
