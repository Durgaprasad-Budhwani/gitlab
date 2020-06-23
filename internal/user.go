package internal

import (
	"net/url"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportRepoUsers(repo *sdk.SourceCodeRepo) error {

	return g.exportUsers(repo, func(user *sdk.SourceCodeUser) error {
		return g.pipe.Write(user)
	})

}

func (g *GitlabIntegration) exportProjectUsers(project *sdk.WorkProject) (usermap api.UsernameMap, rerr error) {

	usermap = make(api.UsernameMap)

	rerr = g.exportUsers(ToRepo(project), func(user *sdk.SourceCodeUser) error {
		usermap[*user.Username] = user.RefID
		return g.pipe.Write(toWorkUser(user))
	})

	return
}

func toWorkUser(user *sdk.SourceCodeUser) *sdk.WorkUser {
	var username string
	if user.Username != nil {
		username = *user.Username
	}
	return &sdk.WorkUser{
		AssociatedRefID: user.AssociatedRefID,
		AvatarURL:       user.AvatarURL,
		CustomerID:      user.CustomerID,
		Email:           user.Email,
		ID:              user.ID,
		Member:          user.Member,
		Name:            user.Name,
		RefID:           user.RefID,
		RefType:         user.RefType,
		URL:             user.URL,
		Username:        username,
		Hashcode:        user.Hashcode,
	}
}

type callBackSourceUser func(item *sdk.SourceCodeUser) error

func (g *GitlabIntegration) exportUsers(repo *sdk.SourceCodeRepo, callback callBackSourceUser) (rerr error) {
	return api.PaginateStartAt(g.logger, "", func(log sdk.Logger, params url.Values) (pi api.PageInfo, err error) {
		params.Set("per_page", MaxFetchedEntitiesCount)
		pi, arr, err := api.RepoUsersPage(g.qc, repo, params)
		if err != nil {
			return
		}
		for _, item := range arr {
			if err = callback(item); err != nil {
				return
			}
		}
		return
	})
}

func (g *GitlabIntegration) exportEnterpriseUsers() error {
	return g.fetchEnterpriseUsers(func(user *sdk.SourceCodeUser) error {
		return g.pipe.Write(user)
	})
}

func (g *GitlabIntegration) fetchEnterpriseUsers(callback callBackSourceUser) (rerr error) {
	return api.PaginateStartAt(g.logger, "", func(log sdk.Logger, params url.Values) (pi api.PageInfo, err error) {
		params.Set("per_page", MaxFetchedEntitiesCount)
		params.Set("membership", "true")
		pi, arr, err := api.UsersPage(g.qc, params)
		if err != nil {
			return
		}
		for _, item := range arr {
			if err = callback(item); err != nil {
				return
			}
		}
		return
	})
}
