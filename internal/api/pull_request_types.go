package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent.next/sdk"
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
	pr.CustomerID = customerID
	pr.RefType = refType
	pr.RefID = prRefID
	pr.RepoID = repoID
	pr.Active = true

	pr.Title = c.Title
	pr.BranchName = c.SourceBranch
	pr.Description = setHTMLPRDescription(c.Description)
	pr.Draft = c.Draft
	switch c.State {
	case "opened":
		pr.Status = sdk.SourceCodePullRequestStatusOpen
	case "closed":
		sdk.ConvertTimeToDateModel(c.ClosedAt, &pr.ClosedDate)
		pr.Status = sdk.SourceCodePullRequestStatusClosed
	case "locked":
		pr.Status = sdk.SourceCodePullRequestStatusLocked
	case "merged":
		pr.MergeSha = c.MergeCommitSHA
		sdk.ConvertTimeToDateModel(c.MergedAt, &pr.MergedDate)
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
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Username  string `json:"username"`
	WebURL    string `json:"web_url"`
}

func (a *author) RefID(customerID string) string {
	return strconv.FormatInt(a.ID, 10)
}

func (a *author) ToModel(customerID string, integrationInstanceID string) *sdk.SourceCodeUser {
	user := &sdk.SourceCodeUser{}
	user.CustomerID = customerID
	user.RefID = a.RefID(customerID)
	user.RefType = "gitlab"
	user.IntegrationInstanceID = sdk.StringPointer(integrationInstanceID)
	user.URL = sdk.StringPointer(a.WebURL)
	user.AvatarURL = sdk.StringPointer(a.AvatarURL)
	// user.Email = Not email for this
	user.Name = a.Name
	var userType sdk.SourceCodeUserType
	if strings.Contains(a.Name, "Bot") {
		userType = sdk.SourceCodeUserTypeBot
	} else {
		userType = sdk.SourceCodeUserTypeHuman
	}

	user.Type = userType
	user.Username = sdk.StringPointer(a.Username)

	return user
}

type apiPullRequest struct {
	*CommonPullRequestFields
	WebURL     string    `json:"web_url"`
	Author     author    `json:"author"`
	ClosedBy   author    `json:"closed_by"`
	MergedBy   author    `json:"merged_by"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
	References struct {
		Full string `json:"full"` // this looks how we display in Gitlab such as !1
	} `json:"references"`
}

func (amr *apiPullRequest) toSourceCodePullRequest(logger sdk.Logger, customerID, repoID string, refType string) (pr *sdk.SourceCodePullRequest) {

	pr = amr.CommonPullRequestFields.commonToSourceCodePullRequest(logger, customerID, repoID, refType)

	sdk.ConvertTimeToDateModel(amr.CreatedAt, &pr.CreatedDate)
	sdk.ConvertTimeToDateModel(amr.UpdatedAt, &pr.UpdatedDate)

	pr.URL = amr.WebURL
	pr.CreatedByRefID = strconv.FormatInt(amr.Author.ID, 10)
	switch amr.State {
	case "closed":
		pr.ClosedByRefID = strconv.FormatInt(amr.ClosedBy.ID, 10)
	case "merged":
		pr.MergedByRefID = strconv.FormatInt(amr.MergedBy.ID, 10)
	}

	pr.Identifier = amr.References.Full

	return
}

// WebhookPullRequest
type WebhookPullRequest struct {
	*CommonPullRequestFields
	URL       string `json:"url"`
	AuthorID  int64  `json:"author_id"`
	UpdatedAt string `json:"updated_at"`
	CreatedAt string `json:"created_at"`
	Action    string `json:"action"`
}

func (wpr *WebhookPullRequest) ToSourceCodePullRequest(logger sdk.Logger, customerID, repoID string, refType string) (pr *sdk.SourceCodePullRequest, err error) {

	pr = wpr.CommonPullRequestFields.commonToSourceCodePullRequest(logger, customerID, repoID, refType)

	createdAt, err := time.Parse("2006-01-02 15:04:05 MST", wpr.CreatedAt)
	if err != nil {
		return
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05 MST", wpr.UpdatedAt)
	if err != nil {
		return
	}

	sdk.ConvertTimeToDateModel(createdAt, &pr.CreatedDate)
	sdk.ConvertTimeToDateModel(updatedAt, &pr.UpdatedDate)

	pr.URL = wpr.URL
	pr.CreatedByRefID = strconv.FormatInt(wpr.AuthorID, 10)
	switch wpr.State {
	case "closed":
		sdk.ConvertTimeToDateModel(updatedAt, &pr.ClosedDate)
	}

	pr.Identifier = fmt.Sprintf("!%d", wpr.IID)

	return
}
