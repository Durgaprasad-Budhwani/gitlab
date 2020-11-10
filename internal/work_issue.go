package internal

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportIssueEntitiesAndWrite(project *api.GitlabProjectInternal, issues chan *sdk.WorkIssue, users api.UsernameMap) {

	var wg sync.WaitGroup

	for issue := range issues {
		wg.Add(1)
		go func(issue *sdk.WorkIssue) {
			defer wg.Done()
			err := ge.exportIssueFields(project, issue, users)
			if err != nil {
				sdk.LogError(ge.logger, "error on issue fields", "err", err)
			}
			issue.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(issue); err != nil {
				sdk.LogError(ge.logger, "error writting issue", "err", err)
			}
		}(issue)
	}

	wg.Wait()

	return
}

func (ge *GitlabExport) exportProjectIssues(project *api.GitlabProjectInternal, users api.UsernameMap) {

	sdk.LogDebug(ge.logger, "exporting project issues", "project", project.Name)

	issuesC := make(chan *sdk.WorkIssue, 10)

	done := make(chan bool, 1)
	go func() {
		ge.exportIssueEntitiesAndWrite(project, issuesC, users)
		done <- true
	}()

	go func() {
		defer close(issuesC)
		var err error
		err = ge.fetchProjectIssues(project, issuesC)
		if err != nil {
			sdk.LogError(ge.logger, "error exporting project issues", "project", project.Name, "err", err)
			done <- true
		}
	}()

	<-done
}

func (ge *GitlabExport) fetchProjectIssues(project *api.GitlabProjectInternal, pissues chan *sdk.WorkIssue) (err error) {
	var nP api.NextPage
	for {
		nP, err = api.WorkIssuesPage(ge.qc, project, nP, pissues)
		if err != nil {
			return err
		}
		if nP == "" {
			return
		}
	}
}

func (ge *GitlabExport) writeSingleIssue(project *sdk.SourceCodeRepo, iid int64) error {

	params := url.Values{}
	params.Set("iids[]", strconv.FormatInt(iid, 10))

	issuesC := make(chan *sdk.WorkIssue, 1)
	_, err := api.WorkSingleIssue(ge.qc, project, ge.lastExportDate, params, issuesC)
	if err != nil {
		return err
	}
	issue := <-issuesC

	issue.IntegrationInstanceID = ge.integrationInstanceID

	return ge.qc.Pipe.Write(issue)
}

func (ge *GitlabExport) createIssue(mutation *sdk.WorkIssueCreateMutation) (*sdk.MutationResponse, error) {

	input, issueType, err := makeCreateMutation(ge.logger, mutation.Fields)
	if err != nil {
		return nil, err
	}

	switch issueType {
	case api.BugIssueType:
		return api.CreateWorkIssue(ge.qc, input, *mutation.Project.RefID, "")
	case api.EnhancementIssueType:
		return api.CreateWorkIssue(ge.qc, input, *mutation.Project.RefID, api.EnhancementIssueType)
	case api.IncidentIssueType:
		return api.CreateWorkIssue(ge.qc, input, *mutation.Project.RefID, api.BugIssueType)
	case api.EpicIssueType:
		return api.CreateEpic(ge.qc, input, *mutation.Project.Name, *mutation.Project.RefID)
	case api.MilestoneIssueType:
		return api.CreateMilestone(ge.qc, input, *mutation.Project.Name, *mutation.Project.RefID)
	default:
		sdk.LogDebug(ge.logger, "issue type not supported", "type", issueType)
	}

	return nil, nil
}

func makeCreateMutation(logger sdk.Logger, fields []sdk.MutationFieldValue) (map[string]interface{}, string, error) {

	params := make(map[string]interface{})

	var issueType string

	for _, fieldVal := range fields {
		switch fieldVal.RefID {
		case "issueType":
			iType, err := getRefID(fieldVal)
			if err != nil {
				return nil, "", fmt.Errorf("error decoding issue type field: %w", err)
			}
			issueType = iType
		case "title":
			title, err := fieldVal.AsString()
			if err != nil {
				return nil, "", fmt.Errorf("error decoding title field: %w", err)
			}
			params["title"] = title
		case "description":
			description, err := fieldVal.AsString()
			if err != nil {
				return nil, "", fmt.Errorf("error decoding description field: %w", err)
			}
			params["description"] = description
		case "dueDate":
			date, err := fieldVal.AsDate()
			if err != nil {
				return nil, "", fmt.Errorf("error decoding due date field: %w", err)
			}

			d := sdk.DateFromEpoch(date.Epoch)

			params["due_date"] = d.Format(api.GitLabDateTimeFormat)
			params["due_date_fixed"] = d.Format(api.GitLabDateTimeFormat)
			params["due_date_is_fixed"] = true
		case "startDate":
			date, err := fieldVal.AsDate()
			if err != nil {
				return nil, "", fmt.Errorf("error decoding start date field: %w", err)
			}

			d := sdk.DateFromEpoch(date.Epoch)

			params["start_date_fixed"] = d.Format(api.GitLabDateTimeFormat)
			params["start_date_is_fixed"] = true

		}
	}
	return params, issueType, nil
}

func getRefID(val sdk.MutationFieldValue) (string, error) {
	nameID, err := val.AsNameRefID()
	if err != nil {
		return "", fmt.Errorf("error decoding %s field as NameRefID: %w", val.Type.String(), err)
	}
	if nameID.RefID == nil {
		return "", errors.New("ref_id was omitted")
	}
	return *nameID.RefID, nil
}
