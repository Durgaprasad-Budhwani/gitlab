package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportProjectSprints(project *sdk.WorkProject) error {
	return api.Paginate(g.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, sprints, err := api.WorkSprintPage(g.qc, project, params)
		if err != nil {
			return pi, err
		}
		for _, s := range sprints {
			if err := g.pipe.Write(s); err != nil {
				return
			}
		}
		return
	})
}
