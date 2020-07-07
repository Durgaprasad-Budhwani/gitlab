package api

import (
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/pkg/util"
	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/datetime"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

type PullRequest struct {
	*sdk.SourceCodePullRequest
	IID           string
	LastCommitSHA string
}

func PullRequestPage(
	qc QueryContext,
	repo *sdk.SourceCodeRepo,
	params url.Values,
	prs chan PullRequest) (pi PageInfo, err error) {

	params.Set("scope", "all")
	params.Set("state", "all")

	sdk.LogDebug(qc.Logger, "repo pull requests", "repo", repo.Name, "repo_id", repo.RefID, "params", params)

	objectPath := pstrings.JoinURL("projects", repo.RefID, "merge_requests")

	var rprs []struct {
		ID           int64     `json:"id"`
		IID          int64     `json:"iid"`
		UpdatedAt    time.Time `json:"updated_at"`
		CreatedAt    time.Time `json:"created_at"`
		ClosedAt     time.Time `json:"closed_at"`
		MergedAt     time.Time `json:"merged_at"`
		SourceBranch string    `json:"source_branch"`
		Title        string    `json:"title"`
		Description  string    `json:"description"`
		WebURL       string    `json:"web_url"`
		State        string    `json:"state"`
		Draft        bool      `json:"work_in_progress"`
		Author       struct {
			ID string `json:"id"`
		} `json:"author"`
		ClosedBy struct {
			ID string `json:"id"`
		} `json:"closed_by"`
		MergedBy struct {
			ID string `json:"id"`
		} `json:"merged_by"`
		MergeCommitSHA string `json:"merge_commit_sha"`
		References     struct {
			Short string `json:"short"` // this looks how we display in Gitlab such as !1
		} `json:"references"`
	}

	pi, err = qc.Request(objectPath, params, &rprs)
	if err != nil {
		return
	}

	repoID := sdk.NewSourceCodeRepoID(qc.CustomerID, repo.RefID, qc.RefType)

	for _, rpr := range rprs {
		prRefID := strconv.FormatInt(rpr.ID, 10)
		pr := &sdk.SourceCodePullRequest{}
		pr.ID = sdk.NewSourceCodePullRequestID(pr.CustomerID, prRefID, qc.RefType, repoID)
		pr.CustomerID = qc.CustomerID
		pr.RefType = qc.RefType
		pr.RefID = prRefID
		pr.RepoID = repoID
		pr.BranchName = rpr.SourceBranch
		// pr.BranchID This needs to be set after getting branch ID
		pr.Title = rpr.Title
		pr.Description = util.ConvertMarkdownToHTML(rpr.Description)
		pr.URL = rpr.WebURL
		pr.Identifier = rpr.References.Short
		datetime.ConvertToModel(rpr.CreatedAt, &pr.CreatedDate)
		datetime.ConvertToModel(rpr.MergedAt, &pr.MergedDate)
		datetime.ConvertToModel(rpr.ClosedAt, &pr.ClosedDate)
		datetime.ConvertToModel(rpr.UpdatedAt, &pr.UpdatedDate)
		switch rpr.State {
		case "opened":
			pr.Status = sdk.SourceCodePullRequestStatusOpen
		case "closed":
			pr.Status = sdk.SourceCodePullRequestStatusClosed
			pr.ClosedByRefID = rpr.ClosedBy.ID
		case "locked":
			pr.Status = sdk.SourceCodePullRequestStatusLocked
		case "merged":
			pr.MergeSha = rpr.MergeCommitSHA
			pr.MergeCommitID = sdk.NewSourceCodePullRequestCommentID(qc.CustomerID, rpr.MergeCommitSHA, qc.RefType, repoID)
			pr.MergedByRefID = rpr.MergedBy.ID
			pr.Status = sdk.SourceCodePullRequestStatusMerged
		default:
			sdk.LogError(qc.Logger, "PR has an unknown state", "state", rpr.State, "ref_id", pr.RefID)
		}
		pr.CreatedByRefID = rpr.Author.ID
		pr.Draft = rpr.Draft

		spr := PullRequest{}
		spr.IID = strconv.FormatInt(rpr.IID, 10)
		spr.SourceCodePullRequest = pr
		prs <- spr
	}

	return
}
