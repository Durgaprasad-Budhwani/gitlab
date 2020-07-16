package api

import (
	"net/url"
	"strconv"

	"github.com/pinpt/agent.next/sdk"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

// UsernameMap map[username]ref_id
type UsernameMap map[string]string

func RepoUsersPage(qc QueryContext, repo *sdk.SourceCodeRepo, params url.Values) (page NextPage, repos []*sdk.SourceCodeUser, err error) {

	sdk.LogDebug(qc.Logger, "users request", "repo", repo.Name, "repo_ref_id", repo.RefID, "params", params)

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
	}

	return
}

type User struct {
	ID        int64
	Email     string
	Username  string
	Name      string
	AvatarURL string
	URL       string
}

func UsersPage(qc QueryContext, params url.Values) (page NextPage, users []*sdk.SourceCodeUser, err error) {

	sdk.LogDebug(qc.Logger, "users request")

	objectPath := pstrings.JoinURL("/users")

	var rawUsers []UserModel

	page, err = qc.Request(objectPath, params, &rawUsers)
	if err != nil {
		return
	}

	for _, user := range rawUsers {
		refID := strconv.FormatInt(user.ID, 10)
		users = append(users, &sdk.SourceCodeUser{
			ID:         sdk.NewSourceCodeUserID(qc.CustomerID, qc.RefType, refID),
			Email:      pstrings.Pointer(user.Email),
			Username:   pstrings.Pointer(user.Username),
			Name:       user.Name,
			RefID:      refID,
			AvatarURL:  pstrings.Pointer(user.AvatarURL),
			URL:        pstrings.Pointer(user.WebURL),
			Type:       sdk.SourceCodeUserTypeHuman,
			Member:     true,
			CustomerID: qc.CustomerID,
			RefType:    qc.RefType,
		})

	}

	return
}

type GitlabUser struct {
	ID   string
	Name string
}

func LoginUser(qc QueryContext) (u *GitlabUser, err error) {

	sdk.LogDebug(qc.Logger, "user request")

	objectPath := pstrings.JoinURL("user")

	var ru struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	_, err = qc.Request(objectPath, nil, &ru)
	if err != nil {
		return
	}

	u = &GitlabUser{}
	u.ID = strconv.FormatInt(ru.ID, 10)
	u.Name = ru.Name

	return
}
