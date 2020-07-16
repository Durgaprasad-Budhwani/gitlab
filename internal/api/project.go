package api

import (
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/datetime"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

func GroupReposPage(qc QueryContext, group *Group, params url.Values, stopOnUpdatedAt time.Time) (page NextPage, repos []*sdk.SourceCodeRepo, err error) {

	params.Set("with_shared", "no")
	params.Set("include_subgroups", "true")

	sdk.LogDebug(qc.Logger, "group repos request", "group_id", group.ID, "group", group.FullPath, "params", params)

	objectPath := pstrings.JoinURL("groups", group.ID, "projects")

	return reposCommonPage(qc, params, stopOnUpdatedAt, objectPath, sdk.SourceCodeRepoAffiliationOrganization)
}

func UserReposPage(qc QueryContext, user *GitlabUser, params url.Values, stopOnUpdatedAt time.Time) (page NextPage, repos []*sdk.SourceCodeRepo, err error) {

	sdk.LogDebug(qc.Logger, "user repos request", "user_id", user.ID, "username", user.Name, "params", params)

	objectPath := pstrings.JoinURL("users", user.ID, "projects")

	return reposCommonPage(qc, params, stopOnUpdatedAt, objectPath, sdk.SourceCodeRepoAffiliationUser)
}

func repoLanguage(qc QueryContext, repoID string) (maxLanguage string, err error) {

	// sdk.LogDebug(qc.Logger, "language request", "repo", repoID)

	objectPath := pstrings.JoinURL("projects", repoID, "languages")

	var languages map[string]float32

	if _, err = qc.Request(objectPath, nil, &languages); err != nil {
		return "", err
	}

	var maxValue float32
	for language, percentage := range languages {
		if percentage > maxValue {
			maxValue = percentage
			maxLanguage = language
		}
	}

	return maxLanguage, nil
}

func reposCommonPage(qc QueryContext, params url.Values, stopOnUpdatedAt time.Time, objectPath string, afiliation sdk.SourceCodeRepoAffiliation) (page NextPage, repos []*sdk.SourceCodeRepo, err error) {

	var rr []struct {
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

	page, err = qc.Request(objectPath, params, &rr)
	if err != nil {
		return
	}

	for _, r := range rr {
		if r.UpdatedAt.Before(stopOnUpdatedAt) {
			return
		}
		refID := strconv.FormatInt(r.ID, 10)
		repo := &sdk.SourceCodeRepo{
			ID:            sdk.NewSourceCodeRepoID(qc.CustomerID, refID, qc.RefType),
			RefID:         refID,
			RefType:       qc.RefType,
			CustomerID:    qc.CustomerID,
			Name:          r.FullName,
			URL:           r.WebURL,
			DefaultBranch: r.DefaultBranch,
			Description:   r.Description,
			UpdatedAt:     datetime.TimeToEpoch(r.UpdatedAt),
			Active:        !r.Archived,
		}

		repo.Language, err = repoLanguage(qc, refID)
		if err != nil {
			return
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

		repos = append(repos, repo)
	}

	return
}
