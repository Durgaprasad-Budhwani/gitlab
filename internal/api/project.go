package api

import (
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/datetime"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

func ReposPage(qc QueryContext, groupName string, params url.Values) (page PageInfo, repos []*sdk.SourceCodeRepo, err error) {

	sdk.LogDebug(qc.Logger, "repos request", "group", groupName)

	objectPath := pstrings.JoinURL("groups", url.QueryEscape(groupName), "projects")

	var rr []struct {
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"last_activity_at"`
		ID            int64     `json:"id"`
		FullName      string    `json:"path_with_namespace"`
		Description   string    `json:"description"`
		WebURL        string    `json:"web_url"`
		Archived      bool      `json:"archived"`
		DefaultBranch string    `json:"default_branch"`
	}

	params.Set("with_shared", "no")

	page, err = qc.Request(objectPath, params, &rr)
	if err != nil {
		return
	}

	for _, repo := range rr {
		refID := strconv.FormatInt(repo.ID, 10)
		repo := &sdk.SourceCodeRepo{
			ID:            sdk.NewSourceCodeRepoID(qc.CustomerID, refID, qc.RefType),
			RefID:         refID,
			RefType:       qc.RefType,
			CustomerID:    qc.CustomerID,
			Name:          repo.FullName,
			URL:           repo.WebURL,
			DefaultBranch: repo.DefaultBranch,
			Description:   repo.Description,
			UpdatedAt:     datetime.TimeToEpoch(repo.UpdatedAt),
			Active:        !repo.Archived,
		}

		repo.Language, err = repoLanguage(qc, refID)
		if err != nil {
			return
		}

		repos = append(repos, repo)
	}

	return
}

func repoLanguage(qc QueryContext, repoID string) (maxLanguage string, err error) {

	sdk.LogDebug(qc.Logger, "language request", "repo", repoID)

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
