package api

import (
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

type issueLink struct {
	IssueID    int64  `json:"id"`
	RefID      int64  `json:"issue_link_id"`
	LinkType   string `json:"link_type"`
	References struct {
		Full string `json:"full"`
	} `json:"references"`
}

func getIssueLinksPage(
	qc QueryContext,
	project *GitlabProjectInternal,
	issueIID string,
	params url.Values) (pi NextPage, linkedIssues []*sdk.WorkIssueLinkedIssues, err error) {

	sdk.LogDebug(qc.Logger, "work issue links", "project", project.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "issues", issueIID, "links")

	var links []issueLink

	pi, err = qc.Get(objectPath, nil, &links)
	if err != nil {
		return
	}

	for _, issueLink := range links {

		issueID := strconv.FormatInt(issueLink.IssueID, 10)

		link2 := &sdk.WorkIssueLinkedIssues{}
		link2.RefID = strconv.FormatInt(issueLink.RefID, 10)
		link2.IssueID = sdk.NewWorkIssueID(qc.CustomerID, issueID, qc.RefType)
		link2.IssueRefID = issueID
		link2.IssueIdentifier = issueLink.References.Full
		link2.ReverseDirection = false

		switch issueLink.LinkType {
		case "is_blocked_by":
			link2.LinkType = sdk.WorkIssueLinkedIssuesLinkTypeBlocks
			link2.ReverseDirection = true
		case "blocks":
			link2.LinkType = sdk.WorkIssueLinkedIssuesLinkTypeBlocks
		case "relates_to":
			link2.LinkType = sdk.WorkIssueLinkedIssuesLinkTypeRelates
		default:
			// we only support default names
			sdk.LogWarn(qc.Logger, "issue link not supported", "issue-iid", issueIID, "type", issueLink.LinkType)
			continue
		}

		linkedIssues = append(linkedIssues, link2)
	}

	return
}

// GetIssueLinks get issue links
func GetIssueLinks(
	qc QueryContext,
	project *GitlabProjectInternal,
	issueIID string) (linkedIssues []sdk.WorkIssueLinkedIssues, err error) {

	err = Paginate(qc.Logger, "", time.Time{}, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (NextPage, error) {
		np, links, err := getIssueLinksPage(qc, project, issueIID, params)
		if err != nil {
			return np, err
		}
		for _, link := range links {
			linkedIssues = append(linkedIssues, *link)
		}

		return np, nil
	})

	return
}
