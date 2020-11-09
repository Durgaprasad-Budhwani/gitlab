package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportRepoMilestones(project *api.GitlabProjectInternal) error {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, err := api.RepoMilestonesPage(ge.qc, project, ge.lastExportDate, params)
		if err != nil {
			return pi, err
		}
		return
	})
}

func (ge *GitlabExport) exportProjectsMilestones(repos []*api.GitlabProjectInternal) error {

	for _, repo := range repos {
		err := ge.exportProjectMilestones(repo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ge *GitlabExport) exportProjectMilestones(project *api.GitlabProjectInternal) error {

	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, err := api.RepoMilestonesPage(ge.qc, project, ge.lastExportDate, params)
		if err != nil {
			return pi, err
		}
		return
	})
}

func (ge *GitlabExport) exportGroupMilestones(namespace *api.Namespace, repos []*api.GitlabProjectInternal) error {

	if namespace.Kind == "user" {
		return nil
	}

	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, err := api.GroupMilestonesPage(ge.qc, namespace, repos, ge.lastExportDate, params)
		if err != nil {
			return pi, err
		}
		return
	})
}

func (ge *GitlabExport) fetchGroupSprints(namespace *api.Namespace) ([]*sdk.AgileSprint, error) {
	return api.GetIterations(ge.qc, namespace)
}
