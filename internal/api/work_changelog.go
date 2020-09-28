package api

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/sdk"
)

type ResourceStateEvents struct {
	ID        int       `json:"id"`
	User      UserModel `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	State     string    `json:"state"`
}

func WorkIssuesDiscussionPage(qc QueryContext, project *sdk.SourceCodeRepo, issueID string, usermap UsernameMap, params url.Values) (pi NextPage, changelogs []*sdk.WorkIssueChangeLog, comments []*sdk.WorkIssueComment, err error) {

	params.Set("notes_filter", "0")
	params.Set("persist_filter", "true")
	params.Set("scope", "all")

	sdk.LogDebug(qc.Logger, "work issues changelog", "project", project.Name, "project_ref_id", project.RefID, "issue", issueID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "issues", issueID, "discussions.json")

	var notes []struct {
		ID    string `json:"id"`
		Notes []struct {
			ID        int       `json:"id"`
			Author    UserModel `json:"author"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			System    bool      `json:"system"`
		} `json:"notes"`
	}
	pi, err = qc.Get(objectPath, params, &notes)
	if err != nil {
		return
	}

	for _, n := range notes {
		for _, nn := range n.Notes {
			if !nn.System {
				comment := &sdk.WorkIssueComment{
					Active:    true,
					RefID:     fmt.Sprint(nn.ID),
					RefType:   qc.RefType,
					UserRefID: usermap[nn.Author.Username],
					IssueID:   issueID,
					ProjectID: project.ID,
					Body:      nn.Body,
				}
				sdk.ConvertTimeToDateModel(nn.CreatedAt, &comment.CreatedDate)
				sdk.ConvertTimeToDateModel(nn.UpdatedAt, &comment.UpdatedDate)
				comments = append(comments, comment)
				continue
			}
			if nn.Body == "changed the description" {
				continue
			}
			changelog := &sdk.WorkIssueChangeLog{
				RefID:  fmt.Sprint(nn.ID),
				UserID: usermap[nn.Author.Username],
			}
			sdk.ConvertTimeToDateModel(nn.CreatedAt, &changelog.CreatedDate)

			if strings.HasPrefix(nn.Body, "assigned to ") {
				// IssueChangeLogFieldAssigneeRefID
				reg := regexp.MustCompile(`@(\w)+`)
				all := reg.FindAllString(nn.Body, 2)
				if len(all) == 0 {
					sdk.LogDebug(qc.Logger, "regex failed, body was: "+nn.Body)
					continue
				}
				toUser := strings.Replace(all[0], "@", "", 1)
				toRefID := usermap[toUser]
				var fromUser string
				var fromRefID string
				if strings.HasPrefix(nn.Body, "and unassigned") {
					fromUser = strings.Replace(all[1], "@", "", 1)
					fromRefID = usermap[fromUser]
				}
				changelog.From = fromRefID
				changelog.FromString = fromUser
				changelog.To = toRefID
				changelog.ToString = toUser
				changelog.Field = sdk.WorkIssueChangeLogFieldAssigneeRefID
			} else if strings.HasPrefix(nn.Body, "unassigned ") {
				reg := regexp.MustCompile(`@(\w)+`)
				all := reg.FindAllString(nn.Body, 1)
				fromUser := strings.Replace(all[0], "@", "", 1)
				fromRefID := usermap[fromUser]
				changelog.From = fromRefID
				changelog.FromString = fromUser
				changelog.Field = sdk.WorkIssueChangeLogFieldAssigneeRefID
			} else if strings.HasPrefix(nn.Body, "changed due date to ") {
				// IssueChangeLogFieldDueDate
				strdate := strings.Replace(nn.Body, "changed due date to ", "", 1)
				changelog.To = strdate
				changelog.ToString = strdate
				changelog.Field = sdk.WorkIssueChangeLogFieldDueDate
			} else if strings.Contains(nn.Body, " epic ") {
				// IssueChangeLogFieldEpicID
				changelog.Field = sdk.WorkIssueChangeLogFieldEpicID
				if strings.HasPrefix(nn.Body, "added to ") {
					to := strings.Replace(nn.Body, "added to epic ", "", 1)
					changelog.To = to
					changelog.ToString = to
				} else if strings.HasPrefix(nn.Body, "changed epic ") {
					to := strings.Replace(nn.Body, "changed epic ", "", 1)
					changelog.To = to
					changelog.ToString = to
				} else if strings.HasPrefix(nn.Body, "removed from ") {
					from := strings.Replace(nn.Body, "removed from epic ", "", 1)
					changelog.From = from
					changelog.FromString = from
				}
			} else if strings.HasPrefix(nn.Body, "changed title") {
				// IssueChangeLogFieldTitle
				reg := regexp.MustCompile(`\*\*(.*?)\*\*`)
				all := reg.FindAllStringSubmatch(nn.Body, -1)
				if len(all) < 2 {
					sdk.LogDebug(qc.Logger, "regex failed, body was: "+nn.Body)
					continue
				}
				from := all[0][1]
				to := all[1][1]
				changelog.From = from
				changelog.FromString = from
				changelog.To = to
				changelog.ToString = to
				changelog.Field = sdk.WorkIssueChangeLogFieldTitle
			} else {
				// not found, continue
				continue
			}
			changelogs = append(changelogs, changelog)

		}
	}

	sdk.LogDebug(qc.Logger, "work issues changelog resource_state_events", "project", project.RefID)

	objectPath = sdk.JoinURL("projects", url.QueryEscape(project.RefID), "issues", issueID, "resource_state_events")

	var stateEvents []ResourceStateEvents
	pi, err = qc.Get(objectPath, params, &stateEvents)
	if err != nil {
		return
	}
	for _, stateEvent := range stateEvents {
		changelog := &sdk.WorkIssueChangeLog{
			RefID:  fmt.Sprint(stateEvent.ID),
			UserID: strconv.FormatInt(stateEvent.User.ID, 10),
		}
		sdk.ConvertTimeToDateModel(stateEvent.CreatedAt, &changelog.CreatedDate)

		if stateEvent.State == "closed" || stateEvent.State == "reopened" {
			changelog.To = stateEvent.State
			changelog.ToString = stateEvent.State
			changelog.Field = sdk.WorkIssueChangeLogFieldStatus
		}
		changelogs = append(changelogs, changelog)
	}

	return
}
