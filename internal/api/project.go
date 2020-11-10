package api

import (
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

// GitlabProjectInternal gitlab project internal
type GitlabProjectInternal struct {
	sdk.SourceCodeRepo
	OwnerRefID int64
}

// GitlabProject gitlab project
type GitlabProject struct {
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"last_activity_at"`
	RefID                int64           `json:"id"`
	FullName             string          `json:"path_with_namespace"`
	Description          string          `json:"description"`
	WebURL               string          `json:"web_url"`
	Archived             bool            `json:"archived"`
	DefaultBranch        string          `json:"default_branch"`
	Visibility           string          `json:"visibility"`
	ForkedFromProject    json.RawMessage `json:"forked_from_project"`
	ApprovalsBeforeMerge json.RawMessage `json:"marked_for_deletion_on"`
	Owner                struct {
		RefID int64 `json:"id"`
	} `json:"owner"`
}

func GroupNamespaceReposPage(qc QueryContext, namespace *Namespace, params url.Values, stopOnUpdatedAt time.Time) (page NextPage, repos []*GitlabProjectInternal, err error) {

	params.Set("with_shared", "false")
	params.Set("include_subgroups", "true")

	sdk.LogDebug(qc.Logger, "group repos request", "namespace_id", namespace.ID, "namespace", namespace.FullPath, "params", sdk.Stringify(params))

	objectPath := sdk.JoinURL("groups", namespace.ID, "projects")

	return reposCommonPage(qc, params, stopOnUpdatedAt, objectPath, sdk.SourceCodeRepoAffiliationOrganization, namespace.Name)
}

func UserReposPage(qc QueryContext, namespace *Namespace, params url.Values, stopOnUpdatedAt time.Time) (page NextPage, repos []*GitlabProjectInternal, err error) {

	sdk.LogDebug(qc.Logger, "user repos request", "namespace_path", namespace.Path, "username", namespace.Name, "params", sdk.Stringify(params))

	objectPath := sdk.JoinURL("users", namespace.Path, "projects")

	return reposCommonPage(qc, params, stopOnUpdatedAt, objectPath, sdk.SourceCodeRepoAffiliationUser, namespace.Name)
}

func reposCommonPage(
	qc QueryContext,
	params url.Values,
	stopOnUpdatedAt time.Time,
	objectPath string,
	afiliation sdk.SourceCodeRepoAffiliation,
	groupName string) (page NextPage, repos []*GitlabProjectInternal, err error) {

	var rr []GitlabProject

	page, err = qc.Get(objectPath, params, &rr)
	if err != nil {
		return
	}

	for _, r := range rr {
		repoRefID := strconv.FormatInt(r.RefID, 10)

		repo := sdk.SourceCodeRepo{
			ID:            sdk.NewSourceCodeRepoID(qc.CustomerID, repoRefID, qc.RefType),
			RefID:         repoRefID,
			RefType:       qc.RefType,
			CustomerID:    qc.CustomerID,
			Name:          r.FullName,
			URL:           r.WebURL,
			DefaultBranch: r.DefaultBranch,
			Description:   r.Description,
			UpdatedAt:     sdk.TimeToEpoch(r.UpdatedAt),
			Active:        !r.Archived,
		}

		if r.Visibility == "private" {
			repo.Visibility = sdk.SourceCodeRepoVisibilityPrivate
		} else {
			repo.Visibility = sdk.SourceCodeRepoVisibilityPublic
		}
		if len(r.ForkedFromProject) > 0 {
			repo.Affiliation = sdk.SourceCodeRepoAffiliationThirdparty
		} else {
			repo.Affiliation = afiliation
		}

		rr := &GitlabProjectInternal{}
		rr.OwnerRefID = r.Owner.RefID
		rr.SourceCodeRepo = repo

		qc.WorkManager.AddProjectDetails(ToProject(rr).ID, &ProjectStateInfo{
			ProjectPath: r.FullName,
			GroupPath:   groupName,
		})

		repos = append(repos, rr)
	}

	return
}

func ProjectByRefID(qc QueryContext, projectRefID int64) (repo *sdk.SourceCodeRepo, err error) {

	objectPath := sdk.JoinURL("projects", strconv.FormatInt(projectRefID, 10))

	var r struct {
		CreatedAt         time.Time       `json:"created_at"`
		UpdatedAt         time.Time       `json:"last_activity_at"`
		ID                int64           `json:"id"`
		FullName          string          `json:"path_with_namespace"`
		Description       string          `json:"description"`
		WebURL            string          `json:"web_url"`
		Archived          bool            `json:"archived"`
		DefaultBranch     string          `json:"default_branch"`
		Visibility        string          `json:"visibility"`
		ForkedFromProject json.RawMessage `json:"forked_from_project"`
	}

	_, err = qc.Get(objectPath, nil, &r)
	if err != nil {
		return
	}

	refID := strconv.FormatInt(r.ID, 10)
	repo = &sdk.SourceCodeRepo{
		ID:            sdk.NewSourceCodeRepoID(qc.CustomerID, refID, qc.RefType),
		RefID:         refID,
		RefType:       qc.RefType,
		CustomerID:    qc.CustomerID,
		Name:          r.FullName,
		URL:           r.WebURL,
		DefaultBranch: r.DefaultBranch,
		Description:   r.Description,
		UpdatedAt:     sdk.TimeToEpoch(r.UpdatedAt),
		Active:        !r.Archived,
	}

	if r.Visibility == "private" {
		repo.Visibility = sdk.SourceCodeRepoVisibilityPrivate
	} else {
		repo.Visibility = sdk.SourceCodeRepoVisibilityPublic
	}
	repo.Affiliation = sdk.SourceCodeRepoAffiliationOrganization

	return
}

func ProjectUser(qc QueryContext, repo *GitlabProjectInternal, userId string) (u *GitlabUser, err error) {

	sdk.LogDebug(qc.Logger, "project user access level", "project_name", repo.Name, "project_id", repo.ID, "user_id", userId)

	objectPath := sdk.JoinURL("projects", repo.RefID, "members", userId)

	_, err = qc.Get(objectPath, nil, &u)
	if err != nil {
		return
	}

	u.StrID = strconv.FormatInt(u.RefID, 10)

	return
}
