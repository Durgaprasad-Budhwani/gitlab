package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportEpics(namespace *api.Namespace, repos []*sdk.SourceCodeRepo, projectUsers api.UsernameMap) (rerr error) {
	if namespace.Kind == "user" {
		return
	}
	return api.Paginate(ge.logger, "", ge.lastExportDate, func(log sdk.Logger, params url.Values, _ time.Time) (api.NextPage, error) {
		if ge.lastExportDateGitlabFormat != "" {
			params.Set("updated_after", ge.lastExportDateGitlabFormat)
		}
		pi, epics, err := api.EpicsPage(ge.qc, namespace, params, repos)
		if err != nil {
			return pi, err
		}
		for _, epic := range epics {

			changelogs, err := ge.fetchEpicIssueDiscussions(namespace, repos, epic, projectUsers)
			if err != nil {
				return pi, err
			}

			epic.ChangeLog = changelogs

			sdk.LogDebug(ge.logger, "writting epic", "epic", epic)
			if err := ge.pipe.Write(epic); err != nil {
				return pi, err
			}
		}
		return pi, nil
	})
}
