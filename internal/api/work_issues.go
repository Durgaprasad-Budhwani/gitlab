package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

type Milestone2 struct {
	ID        string `json:"id"`
	StartDate string `json:"startDate"`
	DueDate   string `json:"dueDate"`
}

type Label2 struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Issue2 struct {
	ID        string `json:"id"`
	IID       string `json:"iid"`
	Assignees struct {
		Edges []struct {
			Node struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"assignees"`
	Author struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"author"`
	Description string `json:"description"`
	Epic        *struct {
		ID string `json:"id"`
	} `json:"epic"`
	Reference string `json:"reference"`
	WebPath   string `json:"webPath"`
	Title     string `json:"title"`
	State     string `json:"state"`
	Weight    *int   `json:"weight"`
	Labels    struct {
		Edges []*Label2 `json:"edges"`
		// Edges []struct{} `json:"edges"`
	} `json:"labels"`
	Type      string    `json:"type"`
	WebURL    string    `json:"webUrl"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ClosedAt  time.Time `json:"closedAt"`
	Iteration struct {
		ID        string `json:"id"`
		DueDate   string `json:"dueDate"`
		StartDate string `json:"startDate"`
	} `json:"iteration"`
	Milestone *Milestone2 `json:"milestone"`
	DueDate   interface{} `json:"dueDate"`
}

const issuesQuery = `query {
	project(fullPath:"%s"){
		issues(first:100,after:"%s"){
			pageInfo{
				hasNextPage
				endCursor
			}
			count
			edges{
			  node{
				id
				iid
				assignees{
				  edges{
					node{
					  id
					  username
					}
				  }
				}
				author{
				  id
				  username
				}
				description
				epic{
				  id
				  iid
				}
				reference(full:true)
				webPath
				title
				state
				weight
				labels{
				  edges{
					node{
					  id
					  title
					}
				  }
				}
				type
				webUrl
				webPath
				createdAt
				updatedAt
				closedAt
				iteration{
				  id
				  startDate
				  dueDate
				}
				milestone{
				  id
				  startDate
				  dueDate
				}
				dueDate
			  }
			}
		}
	}
  }`

type UserModel struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}

func (u *UserModel) ToSourceCodeUser(customerID string) *sdk.SourceCodeUser {

	var userType sdk.SourceCodeUserType
	if strings.Contains(u.Name, "Bot") {
		userType = sdk.SourceCodeUserTypeBot
	} else {
		userType = sdk.SourceCodeUserTypeHuman
	}

	refID := strconv.FormatInt(u.ID, 10)

	user := &sdk.SourceCodeUser{
		Email:      sdk.StringPointer(u.Email),
		Username:   sdk.StringPointer(u.Username),
		Name:       u.Name,
		RefID:      refID,
		AvatarURL:  sdk.StringPointer(u.AvatarURL),
		URL:        sdk.StringPointer(u.WebURL),
		Type:       userType,
		CustomerID: customerID,
		RefType:    "gitlab",
	}

	return user
}

const (
	// OpenColumn open column
	OpenColumn int64 = iota
	// ClosedColumn closed column
	ClosedColumn
)

// OpenedState opened state
const OpenedState = "Opened"

// ClosedState closed state
const ClosedState = "Closed"

// StatesMap states map
var StatesMap = map[string]string{
	"opened": OpenedState,
	"closed": ClosedState,
	"active": OpenedState,
}

// BugIssueType bug issue type
const BugIssueType = "Bug"

// EpicIssueType epic issue type
const EpicIssueType = "Epic"

// IncidentIssueType incident issue type
const IncidentIssueType = "Incident"

// EnhancementIssueType enhancement issue type
const EnhancementIssueType = "Enhancement"

// MilestoneIssueType enhancement issue type
const MilestoneIssueType = "Milestone"

func WorkSingleIssue(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	stopOnUpdatedAt time.Time,
	params url.Values,
	issues chan *sdk.WorkIssue) (pi NextPage, err error) {

	params.Set("scope", "all")
	params.Set("with_labels_details", "true")

	sdk.LogDebug(qc.Logger, "work issues", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "issues")

	var rawissues []IssueModel

	pi, err = qc.Get(objectPath, params, &rawissues)
	if err != nil {
		return
	}

	sdk.LogDebug(qc.Logger, "issues found", "len", len(rawissues))

	for _, rawissue := range rawissues {
		if rawissue.UpdatedAt.Before(stopOnUpdatedAt) {
			return
		}

		issues <- rawissue.ToModel(qc, project.RefID, project.Name)
	}

	return
}

// WorkIssuesPage graphql issue page
func WorkIssuesPage(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	nextPageP NextPage,
	issues chan *sdk.WorkIssue) (nextPage NextPage, err error) {

	sdk.LogDebug(qc.Logger, "work issues", "project", project.Name, "project_ref_id", project.RefID)

	var Data struct {
		Project struct {
			Issues struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Count int `json:"count"`
				Edges []struct {
					Node Issue2 `json:"node"`
				} `json:"edges"`
			} `json:"issues"`
		} `json:"project"`
	}

	query := fmt.Sprintf(issuesQuery, project.Name, nextPageP)

	err = qc.GraphRequester.Query(query, nil, &Data)
	if err != nil {
		return
	}

	sdk.LogDebug(qc.Logger, "issues found", "len", Data.Project.Issues.Count)

	for _, rawissue := range Data.Project.Issues.Edges {
		issue, err := rawissue.Node.ToModel(qc, project)
		if err != nil {
			return nextPage, err
		}
		issues <- issue
	}

	if !Data.Project.Issues.PageInfo.HasNextPage {
		return
	}

	nextPage = NextPage(Data.Project.Issues.PageInfo.EndCursor)

	return
}

func getIssueTypeFromLabels(tags []string, qc QueryContext) (string, string) {
	if len(tags) == 0 {
		return BugIssueType, sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, BugIssueType)
	}

	for _, lbl := range tags {
		switch lbl {
		case strings.ToLower(IncidentIssueType):
			// TODO: Add graphql query to get issue type as we can have issues with this label and not be a legit Incident
			return IncidentIssueType, sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, IncidentIssueType)
		case strings.ToLower(EnhancementIssueType):
			return EnhancementIssueType, sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, EnhancementIssueType)
		}
	}

	return BugIssueType, sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, BugIssueType)
}

type IssueWebHook struct {
	IID int64 `json:"iid"`
}

type GitlabEpic struct {
	RefID int64 `json:"id"`
}

type IssueModel struct {
	ID                 int64         `json:"id"`
	Iid                int64         `json:"iid"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	State              string        `json:"state"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	Labels             []interface{} `json:"labels"`
	Milestone          *Milestone
	Author             UserModel   `json:"author"`
	Assignee           *UserModel  `json:"assignee"`
	UserNotesCount     int         `json:"user_notes_count"`
	MergeRequestsCount int         `json:"merge_requests_count"`
	Upvotes            int         `json:"upvotes"`
	Downvotes          int         `json:"downvotes"`
	DueDate            string      `json:"due_date"`
	Confidential       bool        `json:"confidential"`
	DiscussionLocked   interface{} `json:"discussion_locked"`
	WebURL             string      `json:"web_url"`
	TimeStats          struct {
		TimeEstimate        int         `json:"time_estimate"`
		TotalTimeSpent      int         `json:"total_time_spent"`
		HumanTimeEstimate   interface{} `json:"human_time_estimate"`
		HumanTotalTimeSpent interface{} `json:"human_total_time_spent"`
	} `json:"time_stats"`
	TaskCompletionStatus struct {
		Count          int `json:"count"`
		CompletedCount int `json:"completed_count"`
	} `json:"task_completion_status"`
	HasTasks bool `json:"has_tasks"`
	Links    struct {
		Self       string `json:"self"`
		Notes      string `json:"notes"`
		AwardEmoji string `json:"award_emoji"`
		Project    string `json:"project"`
	} `json:"_links"`
	References struct {
		Short    string `json:"short"`
		Relative string `json:"relative"`
		Full     string `json:"full"`
	} `json:"references"`
	MovedToID    interface{} `json:"moved_to_id"`
	Epic         *GitlabEpic `json:"epic"`
	Weight       *int        `json:"weight"`
	ProjectRefID int64       `json:"project_id"`
}

type IssueCreateModel struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
}

func (i *IssueCreateModel) ToReader() (io.Reader, error) {

	bts, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(bts), nil
}

func (i *IssueModel) ToModel(qc QueryContext, projectRefID string, projectPath string) *sdk.WorkIssue {

	issueRefID := strconv.FormatInt(i.ID, 10)
	issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

	projectID := sdk.NewWorkProjectID(qc.CustomerID, projectRefID, qc.RefType)

	item := &sdk.WorkIssue{}
	item.ID = issueID
	item.Active = true
	item.CustomerID = qc.CustomerID
	item.RefType = qc.RefType
	item.RefID = issueRefID

	if i.Assignee != nil {
		item.AssigneeRefID = fmt.Sprint(i.Assignee.ID)
	}

	item.ReporterRefID = fmt.Sprint(i.Author.ID)
	item.CreatorRefID = fmt.Sprint(i.Author.ID)
	item.Description = i.Description
	if i.Epic != nil {
		epicID := sdk.NewWorkIssueID(qc.CustomerID, strconv.FormatInt(i.Epic.RefID, 10), qc.RefType)
		item.EpicID = sdk.StringPointer(epicID)
		item.ParentID = epicID
	}
	item.Identifier = i.References.Full
	item.ProjectIds = []string{sdk.NewWorkProjectID(qc.CustomerID, projectRefID, qc.RefType)}
	item.Title = i.Title
	item.Status = StatesMap[i.State]
	item.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, qc.RefType, item.Status)
	if i.Weight != nil {
		value := float64(*i.Weight)
		item.StoryPoints = &value
	}

	tags := make([]string, 0)

	for _, label := range i.Labels {
		switch label.(type) {
		case *Label:
			tags = append(tags, label.(*Label).Name)
		case string:
			tags = append(tags, label.(string))
		}
	}

	issueState := IssueStateInfo{
		IID:          strconv.FormatInt(i.Iid, 10),
		ProjectRefID: projectRefID,
	}
	qc.WorkManager.AddIssue(issueID, i.State == strings.ToLower(OpenedState), projectID, i.Labels, i.Milestone, "", i.Assignee, i.Weight, &issueState)

	item.Tags = tags
	item.Type, item.TypeID = getIssueTypeFromLabels(tags, qc)
	item.URL = i.WebURL

	sdk.ConvertTimeToDateModel(i.CreatedAt, &item.CreatedDate)
	sdk.ConvertTimeToDateModel(i.UpdatedAt, &item.UpdatedDate)

	if i.Milestone != nil {
		item.SprintIds = []string{sdk.NewAgileSprintID(qc.CustomerID, strconv.FormatInt(int64(i.Milestone.RefID), 10), qc.RefType)}

		duedate, err := time.Parse("2006-01-02", i.Milestone.DueDate)
		if err != nil {
			duedate = time.Time{}
		}
		sdk.ConvertTimeToDateModel(duedate, &item.PlannedEndDate)

		startdate, err := time.Parse("2006-01-02", i.Milestone.StartDate)
		if err != nil {
			startdate = time.Time{}
		}
		sdk.ConvertTimeToDateModel(startdate, &item.PlannedStartDate)
	}

	return item
}

func (i *Issue2) ToModel(qc QueryContext, project *sdk.SourceCodeRepo) (*sdk.WorkIssue, error) {

	issueRefID := ExtractGraphQLID(i.ID)

	issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

	projectID := sdk.NewWorkProjectID(qc.CustomerID, project.RefID, qc.RefType)

	item := &sdk.WorkIssue{}
	item.ID = issueID
	item.Active = true
	item.CustomerID = qc.CustomerID
	item.RefType = qc.RefType
	item.RefID = issueRefID

	mainAssignee := &UserModel{}
	for _, assignee := range i.Assignees.Edges {
		userID := ExtractGraphQLID(assignee.Node.ID)
		userRefID, err := strconv.Atoi(userID)
		if err != nil {
			return nil, err
		}
		mainAssignee.ID = int64(userRefID)
		item.AssigneeRefID = ExtractGraphQLID(assignee.Node.ID)
		break
	}

	item.ReporterRefID = fmt.Sprint(i.Author.ID)
	item.CreatorRefID = fmt.Sprint(i.Author.ID)
	item.Description = i.Description
	if i.Epic != nil {
		epicRefID := ExtractGraphQLID(i.Epic.ID)
		epicID := sdk.NewWorkIssueID(qc.CustomerID, epicRefID, qc.RefType)
		item.EpicID = sdk.StringPointer(epicID)
		item.ParentID = epicID
	} else if i.Milestone != nil {
		milestoneRefID := ExtractGraphQLID(i.Milestone.ID)
		milestoneID := sdk.NewWorkIssueID(qc.CustomerID, milestoneRefID, qc.RefType)
		item.EpicID = sdk.StringPointer(milestoneID)
		item.ParentID = milestoneID
	}
	item.Identifier = i.Reference
	item.ProjectIds = []string{sdk.NewWorkProjectID(qc.CustomerID, project.RefID, qc.RefType)}
	item.Title = i.Title
	item.Status = StatesMap[i.State]
	item.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, qc.RefType, item.Status)
	if i.Weight != nil {
		value := float64(*i.Weight)
		item.StoryPoints = &value
	}

	tags := make([]string, 0)

	for _, label := range i.Labels.Edges {
		tags = append(tags, label.Title)
	}

	issueState := IssueStateInfo{
		IID:          i.IID,
		ProjectRefID: project.RefID,
	}

	qc.WorkManager.AddIssue2(issueID, i.State == strings.ToLower(OpenedState), projectID, i.Labels.Edges, i.Milestone, ExtractGraphQLID(i.Iteration.ID), mainAssignee, i.Weight, &issueState)

	item.Tags = tags
	item.Type, item.TypeID = getIssueTypeFromLabels(tags, qc)
	item.URL = i.WebURL

	sdk.ConvertTimeToDateModel(i.CreatedAt, &item.CreatedDate)
	sdk.ConvertTimeToDateModel(i.UpdatedAt, &item.UpdatedDate)

	if i.Milestone != nil {

		item.SprintIds = []string{sdk.NewAgileSprintID(qc.CustomerID, ExtractGraphQLID(i.Iteration.ID), qc.RefType)}

		duedate, err := time.Parse("2006-01-02", i.Iteration.DueDate)
		if err != nil {
			duedate = time.Time{}
		}
		sdk.ConvertTimeToDateModel(duedate, &item.PlannedEndDate)

		startdate, err := time.Parse("2006-01-02", i.Iteration.StartDate)
		if err != nil {
			startdate = time.Time{}
		}
		sdk.ConvertTimeToDateModel(startdate, &item.PlannedStartDate)
	}

	return item, nil
}

// CreateWorkIssue create work issue
func CreateWorkIssue(qc QueryContext, mutation *sdk.WorkIssueCreateMutation, label string) (*sdk.MutationResponse, error) {

	sdk.LogDebug(qc.Logger, "create issue", "project_ref_id", mutation.ProjectRefID, "body", sdk.Stringify(mutation))

	objectPath := sdk.JoinURL("projects", mutation.ProjectRefID, "issues")

	issueCreate := convertMutationToGitlabIssue(mutation)
	if label != "" {
		issueCreate.Labels = []string{strings.ToLower(label)}
	}

	reader, err := issueCreate.ToReader()
	if err != nil {
		return nil, err
	}

	var issue IssueModel

	_, err = qc.Post(objectPath, nil, reader, &issue)
	if err != nil {
		return nil, err
	}

	projectID := sdk.NewWorkProjectID(qc.CustomerID, mutation.ProjectRefID, qc.RefType)

	projectDetails := qc.WorkManager.GetProjectDetails(projectID)

	workIssue := issue.ToModel(qc, mutation.ProjectRefID, projectDetails.ProjectPath)

	transition := sdk.WorkIssueTransitions{}
	transition.RefID = ClosedState
	transition.Name = ClosedState

	workIssue.Transitions = []sdk.WorkIssueTransitions{transition}

	err = qc.Pipe.Write(workIssue)
	if err != nil {
		return nil, err
	}

	return &sdk.MutationResponse{
		RefID:    sdk.StringPointer(workIssue.RefID),
		EntityID: sdk.StringPointer(workIssue.ID),
		URL:      sdk.StringPointer(workIssue.URL),
	}, nil

}

func convertMutationToGitlabIssue(m *sdk.WorkIssueCreateMutation) IssueCreateModel {
	return IssueCreateModel{
		Title:       m.Title,
		Description: m.Description,
	}
}

const updateIssueIterationQuery = `mutation {
	issueSetIteration(input:{
	  clientMutationId:"%s",
	  projectPath:"%s",
	  iid:"%s",
	  iterationId:"gid://gitlab/Iteration/%s"
	}) {
	  errors
	  clientMutationId
	  issue{
		id
	  }
	}
  }`

type issueUpdateResponse struct {
	IssueSetIteration struct {
		Errors           []string `json:"errors"`
		ClientMutationID string   `json:"clientMutationId"`
		Issue            struct {
			ID string `json:"id"`
		} `json:"issue"`
	} `json:"issueSetIteration"`
}

func updateIssueIteration(qc QueryContext, mutationID string, issueRefID string) error {

	var response issueUpdateResponse

	issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

	issueD := qc.WorkManager.GetIssueDetails(issueID)

	projectID := sdk.NewWorkProjectID(qc.CustomerID, issueD.ProjectRefID, qc.RefType)

	projectDetails := qc.WorkManager.GetProjectDetails(projectID)

	query := fmt.Sprintf(updateIssueIterationQuery, mutationID, projectDetails.ProjectPath, issueD.IID, mutationID)

	err := qc.GraphRequester.Query(query, nil, &response)
	if err != nil {
		return err
	}

	if len(response.IssueSetIteration.Errors) > 0 {
		errors := strings.Join(response.IssueSetIteration.Errors, ", ")
		return fmt.Errorf("error creating sprint, mutation-id: %s, error %s, issue-id %s, issue-iid %s", response.IssueSetIteration.ClientMutationID, errors, issueID, issueD.IID)
	}

	return nil

}

func updateIssue(qc QueryContext, issueRefID string, params url.Values) (*IssueModel, error) {

	sdk.LogDebug(qc.Logger, "updatig issue", "issue_ref_id", issueRefID, "params", params)

	issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

	issueD := qc.WorkManager.GetIssueDetails(issueID)

	objectPath := sdk.JoinURL("projects", issueD.ProjectRefID, "issues", issueD.IID)

	var issue IssueModel
	if _, err := qc.Put(objectPath, params, strings.NewReader(""), &issue); err != nil {
		return nil, err
	}

	return nil, nil

}

// UpdateIssueFromMutation update issue from mutation
func UpdateIssueFromMutation(qc QueryContext, mutation sdk.Mutation, event *sdk.WorkIssueUpdateMutation) (res *sdk.MutationResponse, err error) {

	issueRefID := mutation.ID()

	params, hasMutation := makeIssueUpdate(event)
	if hasMutation {
		_, err = updateIssue(qc, issueRefID, params)
		if err != nil {
			return nil, err
		}
	}

	return &sdk.MutationResponse{
		RefID:    sdk.StringPointer(mutation.ID()),
		EntityID: sdk.StringPointer(sdk.NewWorkIssueID(mutation.CustomerID(), mutation.ID(), qc.RefType)),
	}, nil
}

func makeIssueUpdate(event *sdk.WorkIssueUpdateMutation) (params url.Values, hasMutation bool) {

	params = url.Values{}

	if event.Set.Title != nil {
		params.Set("title", *event.Set.Title)
		hasMutation = true
	}

	if event.Set.Epic != nil {
		params.Set("epic_id", *event.Set.Epic.RefID)
		hasMutation = true
	} else if event.Unset.Epic {
		params.Set("epic_id", "0")
		hasMutation = true
	}

	if event.Set.AssigneeRefID != nil {
		params.Set("assignee_ids", fmt.Sprintf("%s", *event.Set.AssigneeRefID))
		hasMutation = true
	} else if event.Unset.Assignee {
		params.Set("assignee_ids", "")
		hasMutation = true
	}

	return params, hasMutation
}
