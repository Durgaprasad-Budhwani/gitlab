package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

const attachementQuery = `query {
	issue(id: "gid://gitlab/Issue/%s") {
		designCollection {
		  	designs(first:100, after:"%s"){
				pageInfo{
					endCursor
				}
				edges {
					node {
						id
						filename
						fullPath
						image
						imageV432x230
						versions(first:100){
							nodes{
								sha
								id
							}
						}
					}
				}
			}
		}
		discussions(first:100, after:"%s") {
			pageInfo {
			  	endCursor
			}
			edges {
			  	node {
					createdAt
					notes(first: 100) {
				  		nodes {
							body
							author {
					  			username
					  			id
							}
				  		}
					}
				}
			}
		}
	}
}`

func getIssueAttachmentsPage(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	issueRefID string,
	designPage NextPage,
	discussionPage NextPage) (nextDesign NextPage, nextDiscussion NextPage, attachments []*sdk.WorkIssueAttachments, err error) {

	sdk.LogDebug(qc.Logger, "work issue resource_state_events", "project", project.RefID)

	var GraphQLResponse struct {
		Issue struct {
			DesignCollection struct {
				Designs struct {
					PageInfo struct {
						EndCursor string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node struct {
							RefID         string `json:"id"`
							Filename      string `json:"filename"`
							FullPath      string `json:"fullPath"`
							Image         string `json:"image"`
							ImageV432x230 string `json:"imageV432x230"`
							Versions      struct {
								Nodes []struct {
									ID string `json:"id"`
								} `json:"nodes"`
							} `json:"versions"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"designs"`
			} `json:"designCollection"`
			Discussions struct {
				PageInfo struct {
					EndCursor string `json:"endCursor"`
				} `json:"pageInfo"`
				Edges []struct {
					Node struct {
						CreatedAt time.Time `json:"createdAt"`
						Notes     struct {
							Nodes []struct {
								Body   string `json:"body"`
								Author struct {
									Username string `json:"username"`
									ID       string `json:"id"`
								} `json:"author"`
							} `json:"nodes"`
						} `json:"notes"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"discussions"`
		} `json:"issue"`
	}

	query := fmt.Sprintf(attachementQuery, issueRefID, designPage, discussionPage)

	err = qc.GraphClient.Query(query, nil, &GraphQLResponse)
	if err != nil {
		return
	}

	if len(GraphQLResponse.Issue.Discussions.Edges) == 0 {
		return
	}

	for _, edge := range GraphQLResponse.Issue.DesignCollection.Designs.Edges {

		attachment := &sdk.WorkIssueAttachments{
			RefID:        edge.Node.RefID,
			Name:         edge.Node.Filename,
			URL:          sdk.JoinURL(qc.BaseURL, edge.Node.FullPath),
			ThumbnailURL: edge.Node.ImageV432x230,
			// Size: api doesn't response with this data
			// MimeType: api doesn't response with this data,
		}

		{
			versionID := edge.Node.Versions.Nodes[0].ID
			ind := strings.LastIndexAny(versionID, "/")
			version := versionID[ind+1:]

			for _, edge := range GraphQLResponse.Issue.Discussions.Edges {
				if strings.Contains(edge.Node.Notes.Nodes[0].Body, version) {
					sdk.ConvertTimeToDateModel(edge.Node.CreatedAt, &attachment.CreatedDate)
					userID := edge.Node.Notes.Nodes[0].Author.ID
					ind := strings.LastIndexAny(userID, "/")
					userRefID := userID[ind+1:]
					attachment.UserRefID = userRefID
				}
			}
		}

		attachments = append(attachments, attachment)
	}

	nextDesign = NextPage(GraphQLResponse.Issue.DesignCollection.Designs.PageInfo.EndCursor)

	nextDiscussion = NextPage(GraphQLResponse.Issue.Discussions.PageInfo.EndCursor)

	return
}

// GetIssueAttachments Get Issue Attachments
func GetIssueAttachments(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	issueRefID string) (allAttachments []sdk.WorkIssueAttachments, err error) {

	var designPage, discussionPage NextPage
	var attachments []*sdk.WorkIssueAttachments
	for {
		designPage, discussionPage, attachments, err = getIssueAttachmentsPage(qc, project, issueRefID, designPage, discussionPage)
		if err != nil {
			return
		}
		for _, a := range attachments {
			allAttachments = append(allAttachments, *a)
		}
		if len(attachments) == 0 {
			return
		}
	}
}
