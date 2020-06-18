package api

import (
	"net/url"
	"reflect"
	"strconv"
	"time"

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
	repoRefID string,
	params url.Values,
	prs chan PullRequest) (pi PageInfo, err error) {

	sdk.LogDebug(qc.Logger, "repo pull requests", "repo", repoRefID, "params", params)

	objectPath := pstrings.JoinURL("projects", url.QueryEscape(repoRefID), "merge_requests")
	params.Set("scope", "all")
	params.Set("state", "all")

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
			Username string `json:"username"`
		} `json:"author"`
		ClosedBy struct {
			Username string `json:"username"`
		} `json:"closed_by"`
		MergedBy struct {
			Username string `json:"username"`
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

	repoID := sdk.NewSourceCodeRepoID(qc.CustomerID, repoRefID, qc.RefType)

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
		pr.Description = rpr.Description
		pr.URL = rpr.WebURL
		pr.Identifier = rpr.References.Short
		ConvertToModel(rpr.CreatedAt, &pr.CreatedDate)
		ConvertToModel(rpr.MergedAt, &pr.MergedDate)
		ConvertToModel(rpr.ClosedAt, &pr.ClosedDate)
		ConvertToModel(rpr.UpdatedAt, &pr.UpdatedDate)
		switch rpr.State {
		case "opened":
			pr.Status = sdk.SourceCodePullRequestStatusOpen
		case "closed":
			pr.Status = sdk.SourceCodePullRequestStatusClosed
			pr.ClosedByRefID = rpr.ClosedBy.Username
		case "locked":
			pr.Status = sdk.SourceCodePullRequestStatusLocked
		case "merged":
			pr.MergeSha = rpr.MergeCommitSHA
			pr.MergeCommitID = sdk.NewSourceCodePullRequestCommentID(qc.CustomerID, rpr.MergeCommitSHA, qc.RefType, repoID)
			pr.MergedByRefID = rpr.MergedBy.Username
			pr.Status = sdk.SourceCodePullRequestStatusMerged
		default:
			sdk.LogError(qc.Logger, "PR has an unknown state", "state", rpr.State, "ref_id", pr.RefID)
		}
		pr.CreatedByRefID = rpr.Author.Username
		pr.Draft = rpr.Draft

		spr := PullRequest{}
		spr.IID = strconv.FormatInt(rpr.IID, 10)
		spr.SourceCodePullRequest = pr
		prs <- spr
	}

	return
}

// ConvertToModel will fill dateModel based on passed time
func ConvertToModel(ts time.Time, dateModel interface{}) {
	if ts.IsZero() {
		return
	}

	date, err := datetime.NewDateWithTime(ts)
	if err != nil {
		// this will never happen NewDateWithTime, always returns nil
		panic(err)
	}

	t := reflect.ValueOf(dateModel).Elem()
	t.FieldByName("Rfc3339").Set(reflect.ValueOf(date.Rfc3339))
	t.FieldByName("Epoch").Set(reflect.ValueOf(date.Epoch))
	t.FieldByName("Offset").Set(reflect.ValueOf(date.Offset))
}
