package internal

import (
	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
	"net/url"
	"time"
)

func (ge *GitlabExport) appendProjectLabels(repo *api.GitlabProjectInternal) error {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, err error) {
		pi, labels, err := api.ProjectLabelsPage(ge.qc, repo, params)
		if err != nil {
			return pi, err
		}
		repo.Labels = append(repo.Labels,labels...)
		return
	})
}