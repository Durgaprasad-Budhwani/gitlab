package api

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

const helperIterationTitle = "Pinpoint Iteration Helper"

type GraphQLIteration struct {
	RefID       string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartDate   string    `json:"startDate"`
	DueDate     string    `json:"dueDate"`
	CreatedAt   time.Time `json:"createdAt"`
	State       string    `json:"state"`
	UpdatedAt   time.Time `json:"updatedAt"`
	WebURL      string    `json:"webUrl"`
	WebPath     string    `json:"webPath"`
}

const iterationsQuery = `query {
	group(fullPath:"%s"){
	  iterations(first:100,after:"%s"){
		pageInfo{
		  endCursor
		}
		edges{
		  node{
			id
			title
			description
			startDate
			dueDate
			createdAt
			state
			updatedAt
			webUrl
			webPath
		  }
		}
	  }
	}
  }`

const createIteration = `mutation {
	createIteration(input:{
	  clientMutationId:"%s",
	  title:"%s",
	  description:"%s",
	  groupPath:"%s",
	  startDate:"%s",
	  dueDate:"%s",
	}) {
	  errors
	  iteration {
		id
		title
	  }
	}
  }`

func getIterationsPage(
	qc QueryContext,
	namespace *Namespace,
	iterationPage NextPage) (nextPage NextPage, sprints []*sdk.AgileSprint, err error) {

	sdk.LogDebug(qc.Logger, "group iterations", "namespace", namespace.Name, "page", iterationPage)

	var Data struct {
		Group struct {
			Iterations struct {
				PageInfo struct {
					EndCursor string `json:"endCursor"`
				} `json:"pageInfo"`
				Edges []struct {
					Node GraphQLIteration `json:"node"`
				} `json:"edges"`
			} `json:"iterations"`
		} `json:"group"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Scope            string `json:"scope"`
	}

	query := fmt.Sprintf(iterationsQuery, namespace.Name, iterationPage)

	err = qc.GraphRequester.Query(query, nil, &Data)
	if err != nil {
		if checkPermissionsIssue(qc.Logger, err, fmt.Sprintf("no permissions to get iterations on this group %s", namespace.Name)) {
			return nextPage, sprints, nil
		}
		return
	}

	if Data.Error != "" {
		return nextPage, sprints, fmt.Errorf("error getting iterations, err %s", sdk.Stringify(Data))
	}

	if len(Data.Group.Iterations.Edges) == 0 {
		return
	}

	for _, edge := range Data.Group.Iterations.Edges {

		sprintRefIDStr := ExtractGraphQLID(edge.Node.RefID)

		sprint := &sdk.AgileSprint{}
		sprint.ID = sdk.NewAgileSprintID(qc.CustomerID, sprintRefIDStr, qc.RefType)
		sprint.IntegrationInstanceID = sdk.StringPointer(qc.IntegrationInstanceID)
		sprint.Active = true
		sprint.CustomerID = qc.CustomerID
		sprint.RefType = qc.RefType
		sprint.RefID = sprintRefIDStr

		start, err := time.Parse("2006-01-02", edge.Node.StartDate)
		if err == nil {
			sdk.ConvertTimeToDateModel(start, &sprint.StartedDate)
		} else {
			if edge.Node.StartDate != "" {
				sdk.LogError(qc.Logger, "could not figure out start date, skipping sprint object", "err", err, "start_date", edge.Node.StartDate)
				continue
			}
		}
		end, err := time.Parse("2006-01-02", edge.Node.DueDate)
		if err == nil {
			sdk.ConvertTimeToDateModel(end, &sprint.EndedDate)
		} else {
			if edge.Node.DueDate != "" {
				sdk.LogError(qc.Logger, "could not figure out due date, skipping sprint object", "err", err, "due_date", edge.Node.DueDate)
				continue
			}
		}

		if edge.Node.State == "closed" {
			sdk.ConvertTimeToDateModel(edge.Node.UpdatedAt, &sprint.CompletedDate)
			sprint.Status = sdk.AgileSprintStatusClosed
		} else {
			if !start.IsZero() && start.After(time.Now()) {
				sprint.Status = sdk.AgileSprintStatusFuture
			} else {
				sprint.Status = sdk.AgileSprintStatusActive
			}
		}

		sdk.ConvertTimeToDateModel(edge.Node.UpdatedAt, &sprint.UpdatedDate)

		sprint.Goal = edge.Node.Description
		sprint.Name = edge.Node.Title
		sprint.URL = sdk.StringPointer(edge.Node.WebURL)

		sprints = append(sprints, sprint)

	}

	nextPage = NextPage(Data.Group.Iterations.PageInfo.EndCursor)

	return
}

// GetIterations get iterations
func GetIterations(
	qc QueryContext,
	namespace *Namespace) (allSprints []*sdk.AgileSprint, err error) {

	var nextPage NextPage
	var sprints []*sdk.AgileSprint
	for {
		nextPage, sprints, err = getIterationsPage(qc, namespace, nextPage)
		if err != nil {
			return
		}
		for _, a := range sprints {
			allSprints = append(allSprints, a)
		}
		if len(sprints) == 0 {
			return
		}
	}
}

type createIterationResponse struct {
	CreateIteration *struct {
		MutationID string           `json:"string"`
		Errors     []string         `json:"errors"`
		Iteration  GraphQLIteration `json:"iteration"`
	} `json:"createIteration"`
	Errors []struct {
		Message   string `json:"message"`
		Locations []struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"locations"`
		Path []string `json:"path"`
	} `json:"errors"`
}

// CreateSprint create sprint
func CreateSprint(qc QueryContext, startDate, endDate time.Time, groupName, clientMutationID, sprintName, sprintGoal string, iteration *createIterationResponse) error {

	sdk.LogDebug(qc.Logger, "creating iteration", "group-name", groupName)

	sDate := startDate.Format(GitLabDateFormat)
	eDate := endDate.Format(GitLabDateFormat)

	query := fmt.Sprintf(createIteration, clientMutationID, sprintName, sprintGoal, groupName, sDate, eDate)

	if err := qc.GraphRequester.Query(query, nil, &iteration); err != nil {
		if checkPermissionsIssue(qc.Logger, err, fmt.Sprintf("no permissions to create sprint on this group %s, sprint %s", groupName, sprintName)) {
			return nil
		}
		return err
	}

	if iteration.CreateIteration != nil && len(iteration.CreateIteration.Errors) > 0 {
		return fmt.Errorf("error creating sprint: namespace %s, error %q", groupName, iteration.CreateIteration.Errors)
	}

	if len(iteration.Errors) > 0 {
		return fmt.Errorf("error creating sprint: namespace %s, errors %s", groupName, sdk.Stringify(iteration.Errors))
	}

	return nil
}

// CreateSprintFromMutation create sprint from sprint
func CreateSprintFromMutation(qc QueryContext, mutationID string, mutation *sdk.AgileSprintCreateMutation) (*sdk.MutationResponse, error) {

	sdk.LogDebug(qc.Logger, "create sprint", "project_ref_id", mutation.ProjectRefID)

	if mutation.Name == "" {
		return nil, errors.New("sprint name cannot be empty")
	}

	if mutation.StartDate.Epoch == 0 || mutation.EndDate.Epoch == 0 {
		return nil, errors.New("start date and end date must both be set")
	}

	if len(mutation.IssueRefIDs) > 0 {
		return nil, errors.New("adding issues to a new sprint is not supported yet")
	}

	var iteration createIterationResponse
	{
		startDate := sdk.DateFromEpoch(mutation.StartDate.Epoch)
		endDate := sdk.DateFromEpoch(mutation.EndDate.Epoch)

		// TODO: change FIX_THIS to the value the UI sends
		err := CreateSprint(qc, startDate, endDate, mutationID, "FIX_THIS", mutation.Name, *mutation.Goal, &iteration)
		if err != nil {
			return nil, err
		}

	}

	refID := ExtractGraphQLID(iteration.CreateIteration.Iteration.RefID)

	return &sdk.MutationResponse{
		RefID:    sdk.StringPointer(refID),
		EntityID: sdk.StringPointer(sdk.NewAgileSprintID(qc.CustomerID, refID, qc.RefType)),
	}, nil

}

const updateIterationQuery = `mutation {
	updateIteration(input:{
	  id:"%s",%s
	}) {
	  errors
	  iteration {
		id
		title
	  }
	}
  }`

type updateIterationResponse struct {
	CreateIteration struct {
		Errors    []string         `json:"errors"`
		Iteration GraphQLIteration `json:"iteration"`
	} `json:"updateIteration"`
}

func UpdateSprint(qc QueryContext, mutation sdk.Mutation, event *sdk.AgileSprintUpdateMutation) (*sdk.MutationResponse, error) {

	iterationRefID := mutation.ID()
	subquery, hasMutation, err := makeIterationUpdate(event)
	if err != nil {
		return nil, err
	}

	var iteration updateIterationResponse
	if hasMutation {

		query := fmt.Sprintf(updateIterationQuery, iterationRefID, subquery)

		err := qc.GraphRequester.Query(query, nil, &iteration)
		if err != nil {
			return nil, err
		}
	}
	if len(event.Set.IssueRefIDs) > 0 {
		for _, issueRefID := range event.Set.IssueRefIDs {
			if err := updateIssueIteration(qc, iterationRefID, issueRefID); err != nil {
				return nil, err
			}
		}
	}
	if len(event.Unset.IssueRefIDs) > 0 {
		for _, issueRefID := range event.Unset.IssueRefIDs {

			issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

			issueD := qc.WorkManager.GetIssueDetails(issueID)

			projectID := sdk.NewWorkProjectID(qc.CustomerID, issueD.ProjectRefID, qc.RefType)

			projectDetails := qc.WorkManager.GetProjectDetails(projectID)

			var iterationID string

			if _, err := qc.State.Get(iterationGroupKey(projectDetails.GroupPath), &iterationID); err != nil {
				return nil, err
			}

			if err := updateIssueIteration(qc, iterationID, issueRefID); err != nil {
				return nil, err
			}
		}
	}
	return &sdk.MutationResponse{
		RefID:    sdk.StringPointer(iterationRefID),
		EntityID: sdk.StringPointer(sdk.NewAgileSprintID(mutation.CustomerID(), iterationRefID, qc.RefType)),
	}, nil
}

func makeIterationUpdate(event *sdk.AgileSprintUpdateMutation) (string, bool, error) {

	var hasMutation bool
	var subquery string
	if event.Set.Name != nil {
		subquery += fmt.Sprintf("title:\"%s\",", *event.Set.Name)
		hasMutation = true
	}
	if event.Set.Goal != nil {
		subquery += fmt.Sprintf("description:\"%s\",", *event.Set.Goal)
		hasMutation = true
	}
	if event.Set.StartDate != nil {
		startDate := sdk.DateFromEpoch(event.Set.StartDate.Epoch)
		subquery += fmt.Sprintf("startDate:\"%s\",", startDate.Format("2006-01-02"))
		hasMutation = true
	}
	if event.Set.EndDate != nil {
		endDate := sdk.DateFromEpoch(event.Set.EndDate.Epoch)
		subquery += fmt.Sprintf("dueDate:\"%s\",", endDate.Format("2006-01-02"))
		hasMutation = true
	}
	// TODO: change FIX_THIS to the value the UI sends
	subquery += fmt.Sprintf("groupPath:\"%s\"", "FIX_THIS")
	return subquery, hasMutation, nil
}

// CreateHelperSprintToUnsetIssues create helper sprint to unset issues
func CreateHelperSprintToUnsetIssues(qc QueryContext, namespace *Namespace) error {

	if namespace.Kind == "user" {
		return nil
	}

	groupName := namespace.Name

	startDate := time.Now().Add((time.Hour * 24 * 2) * 365 * 10)
	endDate := startDate.Add(time.Hour * 24)

	var iteration createIterationResponse
	mutationIdentifier := "export_" + groupName + "_" + time.Now().Format("2006-01-02T15_04_05Z07_00")

	var iterationID string
	ok, err := qc.State.Get(iterationGroupKey(namespace.Name), &iterationID)
	if err != nil {
		return err
	}
	if !ok {
		if err := CreateSprint(qc, startDate, endDate, groupName, mutationIdentifier, helperIterationTitle, "iteration helper to unset issues", &iteration); err != nil {
			if strings.Contains(err.Error(), "Title already being used for another group or project iteration") {
				// get the iteration id
				iteration, err := IterationByTitle(qc, namespace.Name, helperIterationTitle)
				if err != nil {
					return err
				}
				err = qc.State.Set(iterationGroupKey(namespace.Name), strconv.FormatInt(iteration.RefID, 10))
				if err != nil {
					return err
				}
				return nil
			}
			return err
		}
		iterationKey := iterationGroupKey(groupName)

		if iteration.CreateIteration != nil {
			return qc.State.Set(iterationKey, ExtractGraphQLID(iteration.CreateIteration.Iteration.RefID))
		}
	}

	return nil
}

func iterationGroupKey(groupName string) string {
	return fmt.Sprintf("iteration_group_helper_id_%s", groupName)
}

type restIteration struct {
	RefID int64 `json:"id"`
}

// IterationByTitle get iteration by title
func IterationByTitle(qc QueryContext, group, title string) (*restIteration, error) {

	sdk.LogDebug(qc.Logger, "iteartion by title", "title", title)

	params := url.Values{}
	params.Set("search", title)

	objectPath := sdk.JoinURL("groups", url.QueryEscape(group), "iterations")

	var ri []*restIteration

	_, err := qc.Get(objectPath, params, &ri)
	if err != nil {
		return nil, err
	}

	if len(ri) > 1 {
		return nil, fmt.Errorf("It should exist only one Pinpoint iteration helper for this group %s", group)
	}

	if len(ri) == 0 {
		return nil, fmt.Errorf("It should find at least one iteration helper for this group %s", group)
	}

	return ri[0], nil
}
