package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/gitlab/internal/api"
	"github.com/pinpt/agent/sdk"
)

func (ge *GitlabExport) exportGroupBoards(namespace *api.Namespace, repos []*sdk.SourceCodeRepo) error {
	if namespace.Kind == "user" {
		return nil
	}

	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, err := api.GroupBoardsPage(ge.qc, namespace, repos, params)
		if err != nil {
			return pi, err
		}
		return
	})
}

func (ge *GitlabExport) exportReposBoards(repos []*sdk.SourceCodeRepo) error {

	sdk.LogInfo(ge.qc.Logger, "exporting repo boards", "repos", repos)

	for _, repo := range repos {
		err := api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
			pi, err := api.RepoBoardsPage(ge.qc, repo, params)
			if err != nil {
				return pi, err
			}
			return
		})
		if err != nil {
			return err
		}
	}

	return nil
}
