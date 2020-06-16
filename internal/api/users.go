package api

import (
	"net/url"
	"strconv"

	"github.com/pinpt/agent.next/sdk"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

// UsernameMap map[username]ref_id
type UsernameMap map[string]string

func RepoUsersPage(qc QueryContext, repo *sdk.SourceCodeRepo, params url.Values) (page PageInfo, repos []*sdk.SourceCodeUser, err error) {

	sdk.LogDebug(qc.Logger, "users request", "repo", repo, "params", params)

	objectPath := pstrings.JoinURL("projects", url.QueryEscape(repo.RefID), "users")

	var ru []struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		WebURL    string `json:"web_url"`
	}
	page, err = qc.Request(objectPath, params, &ru)
	if err != nil {
		return
	}
	for _, user := range ru {
		sourceUser := sdk.SourceCodeUser{}
		sourceUser.RefType = qc.RefType
		// sourceUser.Email = // No email info here
		sourceUser.CustomerID = qc.CustomerID
		sourceUser.RefID = strconv.FormatInt(user.ID, 10)
		sourceUser.Name = user.Name
		sourceUser.AvatarURL = pstrings.Pointer(user.AvatarURL)
		sourceUser.Username = pstrings.Pointer(user.Username)
		sourceUser.Member = true
		sourceUser.Type = sdk.SourceCodeUserTypeHuman
		sourceUser.URL = pstrings.Pointer(user.WebURL)

		repos = append(repos, &sourceUser)
		// usermap[user.Username] = sourceUser.RefID
	}

	return
}
