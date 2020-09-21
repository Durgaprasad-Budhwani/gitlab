package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportRepoSprints(project *sdk.SourceCodeRepo) error {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, sprints, err := api.RepoSprintsPage(ge.qc, project, ge.lastExportDate, params)
		if err != nil {
			return pi, err
		}
		for _, s := range sprints {
			s.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(s); err != nil {
				return
			}
		}
		return
	})
}

func (ge *GitlabExport) fetchProjectsSprints(repos []*sdk.SourceCodeRepo) ([]*sdk.AgileSprint, error) {

	allSprints := make([]*sdk.AgileSprint, 0)

	for _, repo := range repos {
		sprints, err := ge.fetchProjectSprints(repo)
		if err != nil {
			return nil, err
		}
		allSprints = append(allSprints, sprints...)
	}

	return allSprints, nil
}

func (ge *GitlabExport) fetchProjectSprints(project *sdk.SourceCodeRepo) ([]*sdk.AgileSprint, error) {

	allSprints := make([]*sdk.AgileSprint, 0)

	return allSprints, api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, sprints, err := api.RepoSprintsPage(ge.qc, project, ge.lastExportDate, params)
		if err != nil {
			return pi, err
		}
		for _, s := range sprints {
			s.IntegrationInstanceID = ge.integrationInstanceID
			allSprints = append(allSprints, s)
		}
		return
	})
}

func (ge *GitlabExport) fetchGroupSprints(namespace *api.Namespace) ([]*sdk.AgileSprint, error) {

	if namespace.Kind == "user" {
		return nil, nil
	}

	allSprints := make([]*sdk.AgileSprint, 0)

	return allSprints, api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, sprints, err := api.GroupSprintsPage(ge.qc, namespace, ge.lastExportDate, params)
		if err != nil {
			return pi, err
		}
		for _, s := range sprints {
			s.IntegrationInstanceID = ge.integrationInstanceID
			allSprints = append(allSprints, s)
		}
		return
	})
}
