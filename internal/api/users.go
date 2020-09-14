package api

import (
	"net/url"
	"strconv"
	"strings"

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

	for _, ruser := range rawUsers {
		user := ruser.ToSourceCodeUser(qc.CustomerID)
		users = append(users, user)
	}

	return
}

func UserByID(qc QueryContext, userID int64) (user *sdk.SourceCodeUser, err error) {

	sdk.LogDebug(qc.Logger, "users request")

	objectPath := sdk.JoinURL("/users", strconv.FormatInt(userID, 10))

	var rawUser UserModel

	_, err = qc.Get(objectPath, nil, &rawUser)
	if err != nil {
		return
	}

	user = rawUser.ToSourceCodeUser(qc.CustomerID)

	return
}

type GitlabUser struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Username    string `json:"username"`
	IsAdmin     bool   `json:"is_admin"`
	AccessLevel int64  `json:"access_level"`
	StrID       string
}

func LoginUser(qc QueryContext) (u *GitlabUser, err error) {

	sdk.LogDebug(qc.Logger, "user request")

	objectPath := sdk.JoinURL("user")

	_, err = qc.Get(objectPath, nil, &u)
	if err != nil {
		return nil, err
	}

	u.StrID = strconv.FormatInt(u.ID, 10)

	return
}

type GitUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatarUrl"`
	Username string `json:"username"`
}

func (a *GitUser) ToModel(customerID string, integrationInstanceID string) *sdk.SourceCodeUser {
	user := &sdk.SourceCodeUser{}
	user.CustomerID = customerID
	user.RefID = a.RefID(customerID)
	user.RefType = "gitlab"
	if a.Email != "" {
		id := sdk.Hash(customerID, a.Email)
		if id != user.RefID {
			user.AssociatedRefID = sdk.StringPointer(id)
		}
	}
	user.IntegrationInstanceID = sdk.StringPointer(integrationInstanceID)
	user.URL = sdk.StringPointer("")
	user.AvatarURL = sdk.StringPointer(a.Avatar)
	user.Email = sdk.StringPointer(a.Email)
	user.Name = a.Name
	var userType sdk.SourceCodeUserType
	if strings.Contains(a.Name, "Bot") {
		userType = sdk.SourceCodeUserTypeBot
	} else {
		userType = sdk.SourceCodeUserTypeHuman
	}

	user.Type = userType
	user.Username = sdk.StringPointer("")

	return user
}

func (a *GitUser) RefID(customerID string) string {
	if a.Email != "" {
		return sdk.Hash(customerID, a.Email)
	}
	return ""
}
