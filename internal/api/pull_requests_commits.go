package api

import (
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

// CommonCommitFields common fields on commit objects
type CommonCommitFields struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func (cc *CommonCommitFields) commonToSourceCodeCommit(customerID, refType, repoID, pullRequestID string) (scc *sdk.SourceCodePullRequestCommit) {

	scc = &sdk.SourceCodePullRequestCommit{}
	scc.CustomerID = customerID
	scc.RefType = refType
	scc.RefID = cc.ID
	scc.RepoID = repoID
	scc.PullRequestID = pullRequestID
	scc.Active = true
	scc.Sha = cc.ID
	scc.Message = cc.Message

	return
}

// PrCommit commit specific fields for pr objects
type PrCommit struct {
	*CommonCommitFields
	CreatedAt      time.Time `json:"created_at"`
	AuthorEmail    string    `json:"author_email"`
	AuthorName     string    `json:"author_name"`
	CommitterEmail string    `json:"committer_email"`
	CommitterName  string    `json:"committer_name"`
	WebURL         string    `json:"web_url"`
}

// ToSourceCodePullRequestCommit convert pr commit to source code pr commit
func (prc *PrCommit) ToSourceCodePullRequestCommit(customerID, refType, repoID, pullRequestID string) (scc *sdk.SourceCodePullRequestCommit) {

	scc = prc.CommonCommitFields.commonToSourceCodeCommit(customerID, refType, repoID, pullRequestID)

	scc.URL = prc.WebURL
	scc.AuthorRefID = CodeCommitEmail(customerID, prc.AuthorEmail)
	scc.CommitterRefID = CodeCommitEmail(customerID, prc.CommitterEmail)
	sdk.ConvertTimeToDateModel(prc.CreatedAt, &scc.CreatedDate)

	return
}

// WhCommit commit specific fields for commins in push events
type WhCommit struct {
	*CommonCommitFields
	URL    string `json:"url"`
	Author struct {
		// Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
	Timestamp time.Time `json:"timestamp"`
}

func (wc *WhCommit) ToSourceCodePullRequestCommit(customerID, refType, repoID, pullRequestID string) (scc *sdk.SourceCodePullRequestCommit) {

	scc = wc.CommonCommitFields.commonToSourceCodeCommit(customerID, refType, repoID, pullRequestID)

	scc.URL = wc.URL
	scc.AuthorRefID = CodeCommitEmail(customerID, wc.Author.Email)
	sdk.ConvertTimeToDateModel(wc.Timestamp, &scc.CreatedDate)

	return
}

// PullRequestCommitsPage pr commits page
func PullRequestCommitsPage(
	qc QueryContext,
	repo *GitlabProjectInternal,
	pr PullRequest,
	params url.Values,
	after time.Time) (pi NextPage, res []*sdk.SourceCodePullRequestCommit, err error) {

	sdk.LogDebug(qc.Logger, "pull request commits", "repo", repo.Name, "repo_ref_id", repo.RefID, "pr_iid", pr.IID, "params", params)

	objectPath := sdk.JoinURL("projects", repo.RefID, "merge_requests", pr.IID, "commits")

	var rcommits []PrCommit

	pi, err = qc.Get(objectPath, params, &rcommits)
	if err != nil {
		return
	}

	repoID := sdk.NewSourceCodeRepoID(qc.CustomerID, repo.RefID, qc.RefType)
	pullRequestID := sdk.NewSourceCodePullRequestID(qc.CustomerID, pr.RefID, qc.RefType, repoID)

	for _, rcommit := range rcommits {
		if !after.IsZero() && rcommit.CreatedAt.Before(after) {
			return
		}

		author := commitAuthorUserToAuthor(&rcommit)
		err = qc.UserManager.EmitGitUser(qc.Logger, author)
		if err != nil {
			return
		}

		author = commitCommiterUserToAuthor(&rcommit)
		err = qc.UserManager.EmitGitUser(qc.Logger, author)
		if err != nil {
			return
		}

		item := rcommit.ToSourceCodePullRequestCommit(qc.CustomerID, qc.RefType, repoID, pullRequestID)
		res = append(res, item)
	}

	return
}

// CodeCommitEmail identifier for users using email
func CodeCommitEmail(customerID string, email string) string {
	return sdk.Hash(customerID, email)
}

func commitAuthorUserToAuthor(commit *PrCommit) *GitUser {
	author := &GitUser{}
	author.Email = commit.AuthorEmail
	author.Name = commit.AuthorName
	return author
}

func commitCommiterUserToAuthor(commit *PrCommit) *GitUser {
	author := &GitUser{}
	author.Email = commit.CommitterEmail
	author.Name = commit.CommitterName
	return author
}
