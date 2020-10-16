package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

type Iteration struct {
	ID          string    `json:"id"`
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
					Node Iteration `json:"node"`
				} `json:"edges"`
			} `json:"iterations"`
		} `json:"group"`
	}

	query := fmt.Sprintf(iterationsQuery, namespace.Name, iterationPage)

	err = qc.GraphRequester.Query(query, nil, &Data)
	if err != nil {
		if strings.Contains(err.Error(), "The resource that you are attempting to access does not exist or you don't have permission to perform this action") {
			sdk.LogWarn(qc.Logger, err.Error())
			return nextPage, sprints, nil
		}
		return
	}

	if len(Data.Group.Iterations.Edges) == 0 {
		return
	}

	for _, edge := range Data.Group.Iterations.Edges {

		sprintRefIDStr := ExtractGraphQLID(edge.Node.ID)

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
	CreateIteration struct {
		MutationID string    `json:"string"`
		Errors     []string  `json:"errors"`
		Iteration  Iteration `json:"iteration"`
	} `json:"createIteration"`
}

// CreateSprint create sprint
func CreateSprint(qc QueryContext, mutationID string, mutation *sdk.AgileSprintCreateMutation) (*sdk.MutationResponse, error) {

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

		// TODO: change premium_group2 to be dynamic
		query := fmt.Sprintf(createIteration, mutationID, mutation.Name, *mutation.Goal, "premium_group2", startDate, endDate)

		if err := qc.GraphRequester.Query(query, nil, &iteration); err != nil {
			return nil, err
		}

		if len(iteration.CreateIteration.Errors) > 0 {
			errors := strings.Join(iteration.CreateIteration.Errors, ", ")
			return nil, fmt.Errorf("error creating sprint, mutation-id: %s, error %s, body %v", iteration.CreateIteration.MutationID, errors, sdk.Stringify(mutation))
		}
	}

	refID := ExtractGraphQLID(iteration.CreateIteration.Iteration.ID)

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
		Errors    []string  `json:"errors"`
		Iteration Iteration `json:"iteration"`
	} `json:"updateIteration"`
}

func UpdateSprint(qc QueryContext, mutation sdk.Mutation, event *sdk.AgileSprintUpdateMutation) (*sdk.MutationResponse, error) {

	refID := mutation.ID()
	subquery, hasMutation, err := makeIterationUpdate(event)
	if err != nil {
		return nil, err
	}

	var iteration updateIterationResponse
	if hasMutation {

		query := fmt.Sprintf(updateIterationQuery, refID, subquery)

		err := qc.GraphRequester.Query(query, nil, &iteration)
		if err != nil {
			return nil, err
		}
	}
	if len(event.Set.IssueRefIDs) > 0 {
		for _, issueRefID := range event.Set.IssueRefIDs {
			if err := updateIssueIteration(qc, refID, issueRefID); err != nil {
				return nil, err
			}
		}
	}
	if len(event.Unset.IssueRefIDs) > 0 {
		for _, issueRefID := range event.Unset.IssueRefIDs {
			// TODO: create a dump iteration and change 2288 with that
			if err := updateIssueIteration(qc, "2288", issueRefID); err != nil {
				return nil, err
			}
		}
	}
	return &sdk.MutationResponse{
		RefID:    sdk.StringPointer(refID),
		EntityID: sdk.StringPointer(sdk.NewAgileSprintID(mutation.CustomerID(), refID, qc.RefType)),
	}, nil
}

func makeIterationUpdate(event *sdk.AgileSprintUpdateMutation) (string, bool, error) {

	fmt.Println("event", sdk.Stringify(event))

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
	// TODO: change premium_group2 and make it dynamic
	subquery += fmt.Sprintf("groupPath:\"%s\"", "premium_group2")
	return subquery, hasMutation, nil
}
