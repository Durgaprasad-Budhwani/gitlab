package internal

import (
	"net/url"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// IssueFuture issues will process later
type IssueFuture struct {
	Project *sdk.WorkProject
	Page    api.PageInfo
}

func (g *GitlabIntegration) exportProjectIssues(project *sdk.WorkProject, users api.UsernameMap) error {

	sdk.LogDebug(g.logger, "issues", "project", project.Name)

	page, issues, err := g.fetchInitialProjectIssues(project)
	if err != nil {
		return err
	}

	g.addIssueFuture(project, page)

	return g.exportIssueEntitiesAndWrite(project, issues, users)
}

func (g *GitlabIntegration) fetchInitialProjectIssues(project *sdk.WorkProject) (pi api.PageInfo, res []*sdk.WorkIssue, rerr error) {
	params := url.Values{}
	params.Set("per_page", "10")

	return api.WorkIssuesPage(g.qc, project, params)
}

func (g *GitlabIntegration) addIssueFuture(project *sdk.WorkProject, page api.PageInfo) {
	if page.NextPage != "" {
		g.isssueFutures = append(g.isssueFutures, IssueFuture{project, page})
	}
}

func (g *GitlabIntegration) exportIssueEntitiesAndWrite(project *sdk.WorkProject, issues []*sdk.WorkIssue, users api.UsernameMap) (err error) {
	for _, issue := range issues {
		err = g.exportIssueDiscussions(project, issue, users)
		if err != nil {
			return err
		}
	}

	return g.writeIssues(issues)
}

func (g *GitlabIntegration) writeIssues(issues []*sdk.WorkIssue) (rerr error) {
	for _, issue := range issues {
		if err := g.pipe.Write(issue); err != nil {
			return err
		}
	}
	return
}

func (g *GitlabIntegration) exportRemainingProjectIssues(project *sdk.WorkProject, users api.UsernameMap) error {

	sdk.LogDebug(g.logger, "remaining issues", "project", project.Name)

	issues, err := g.fetchRemainingProjectIssues(project)
	if err != nil {
		return err
	}

	return g.exportIssueEntitiesAndWrite(project, issues, users)
}

func (g *GitlabIntegration) fetchRemainingProjectIssues(project *sdk.WorkProject) (issues []*sdk.WorkIssue, rerr error) {
	rerr = api.PaginateStartAt(g.logger, "2", func(log sdk.Logger, params url.Values) (pi api.PageInfo, rerr error) {
		params.Set("per_page", MaxFetchedEntitiesCount)
		pi, issues, rerr := api.WorkIssuesPage(g.qc, project, params)
		if rerr != nil {
			return
		}
		for _, issue := range issues {
			issues = append(issues, issue)
		}
		return
	})
	return
}
