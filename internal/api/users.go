package api

import (
	"net/url"
	"strconv"

	"github.com/pinpt/agent.next/sdk"
)

const (
	NoAccess   = 0
	Guest      = 10
	Reporter   = 20
	Developer  = 30
	Maintainer = 40
	Owner      = 50
)

// UsernameMap map[username]ref_id
type UsernameMap map[string]string

func RepoUsersPage(qc QueryContext, repo *sdk.SourceCodeRepo, params url.Values) (page NextPage, repos []*sdk.SourceCodeUser, err error) {

	sdk.LogDebug(qc.Logger, "users request", "repo", repo.Name, "repo_ref_id", repo.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(repo.RefID), "users")

	var ru []struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		WebURL    string `json:"web_url"`
	}
	page, err = qc.Get(objectPath, params, &ru)
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
		sourceUser.AvatarURL = sdk.StringPointer(user.AvatarURL)
		sourceUser.Username = sdk.StringPointer(user.Username)
		sourceUser.Member = true
		sourceUser.Type = sdk.SourceCodeUserTypeHuman
		sourceUser.URL = sdk.StringPointer(user.WebURL)

		sdk.StringPointer()

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

	objectPath := sdk.JoinURL("/users")

	var rawUsers []UserModel

	page, err = qc.Get(objectPath, params, &rawUsers)
	if err != nil {
		return
	}

	for _, user := range rawUsers {
		refID := strconv.FormatInt(user.ID, 10)
		users = append(users, &sdk.SourceCodeUser{
			ID:         sdk.NewSourceCodeUserID(qc.CustomerID, qc.RefType, refID),
			Email:      sdk.StringPointer(user.Email),
			Username:   sdk.StringPointer(user.Username),
			Name:       user.Name,
			RefID:      refID,
			AvatarURL:  sdk.StringPointer(user.AvatarURL),
			URL:        sdk.StringPointer(user.WebURL),
			Type:       sdk.SourceCodeUserTypeHuman,
			Member:     true,
			CustomerID: qc.CustomerID,
			RefType:    qc.RefType,
		})

	}

	return
}

type GitlabUser struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	IsAdmin     bool   `json:"is_admin"`
	AccessLevel int64  `json:"access_level"`
	StrID       string
}

func LoginUser(qc QueryContext) (u *GitlabUser, err error) {

	sdk.LogDebug(qc.Logger, "user request")

	objectPath := sdk.JoinURL("user")

	_, err = qc.Get(objectPath, nil, &u)
	if err != nil {
		return
	}

	u.StrID = strconv.FormatInt(u.ID, 10)

	return
}
