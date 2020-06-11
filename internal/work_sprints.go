package internal

import (
	"net/url"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportProjectSprints(project *sdk.WorkProject) error {
	return api.PaginateStartAt(g.logger, "", func(log sdk.Logger, params url.Values) (pi api.PageInfo, rerr error) {
		params.Set("per_page", MaxFetchedEntitiesCount)
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

// err = api.PaginateStartAt(s.logger, func(log hclog.Logger, paginationParams url.Values) (page api.PageInfo, _ error) {

// 		if !lastUpdated.IsZero() {
// 			paginationParams.Set("updated_after", lastUpdated.Format(time.RFC3339Nano))
// 		}
// 		pi, res, err := api.WorkSprintPage(s.qc, proj.GetID(), paginationParams)
// 		if err != nil {
// 			return pi, err
// 		}

// 		if err = projectSender.SetTotal(pi.Total); err != nil {
// 			return pi, err
// 		}
// 		for _, obj := range res {
// 			s.logger.Info("sending sprint", "sprint", obj.RefID)
// 			err := projectSender.Send(obj)
// 			if err != nil {
// 				return pi, err
// 			}
// 		}
// 		return pi, nil
// 	})
// 	if err != nil {
// 		return err
