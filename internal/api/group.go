package api

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

// Namespace internal namespace
type Namespace struct {
	ID                            string
	Name                          string
	Path                          string `json:"path"`
	FullPath                      string
	ValidTier                     bool `json:"valid_tier"`
	MarkedToCreateProjectWebHooks bool
	Visibility                    string
	AvatarURL                     string
	Kind                          string
}

// AllNamespaces all namespaces
func AllNamespaces(qc QueryContext) (allnamespaces []*Namespace, err error) {
	err = Paginate(qc.Logger, "", time.Time{}, func(log sdk.Logger, paginationParams url.Values, t time.Time) (np NextPage, _ error) {
		paginationParams.Set("top_level_only", "true")

		pi, namespaces, err := NamespacesPage(qc, paginationParams)
		if err != nil {
			return pi, err
		}
		allnamespaces = append(allnamespaces, namespaces...)
		return pi, nil
	})
	return
}

func AllNamespaces2(qc *QueryContext2,logger sdk.Logger) (allnamespaces []*GitlabNamespace, err error) {
	err = Paginate2("", time.Time{}, func( paginationParams url.Values, t time.Time) (np NextPage, _ error) {
		paginationParams.Set("top_level_only", "true")

		pi, namespaces, err := NamespacesPage2(qc,logger, paginationParams)
		if err != nil {
			return pi, err
		}
		allnamespaces = append(allnamespaces, namespaces...)
		return pi, nil
	})
	return
}

type rawNamespace struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
	Path     string `json:"path"`
	ParentID int64  `json:"parent_id"`
	// If we have this field it means there is a valid tier
	MembersCountWithDescendants json.RawMessage `json:"members_count_with_descendants"`
	AvatarURL                   string          `json:"avatar_url"`
	Kind                        string          `json:"kind"`
}

func (g *rawNamespace) reset() {
	g.ID = 0
	g.Name = ""
	g.FullPath = ""
	g.MembersCountWithDescendants = []byte("")
	g.AvatarURL = ""
	g.Kind = ""
	g.Path = ""
	g.ParentID = 0
}

// Namespaces fetch namespaces
func NamespacesPage(qc QueryContext, params url.Values) (np NextPage, namespaces []*Namespace, err error) {

	sdk.LogDebug(qc.Logger, "namespaces request", "params", sdk.Stringify(params))

	objectPath := "namespaces"

	var rawNamespaces []json.RawMessage

	np, err = qc.Get(objectPath, params, &rawNamespaces)
	if err != nil {
		return
	}

	var namespace rawNamespace

	for _, n := range rawNamespaces {
		err = json.Unmarshal(n, &namespace)
		if err != nil {
			return
		}

		// Skip subgroups
		if namespace.ParentID != 0 {
			namespace.reset()
			continue
		}

		if !strings.Contains(namespace.AvatarURL, "https") && namespace.AvatarURL != "" {
			namespace.AvatarURL = qc.BaseURL + namespace.AvatarURL
		}

		namespaces = append(namespaces, &Namespace{
			ID:        strconv.FormatInt(namespace.ID, 10),
			Name:      namespace.Name,
			FullPath:  namespace.FullPath,
			ValidTier: isValidTier(n),
			AvatarURL: namespace.AvatarURL,
			Kind:      namespace.Kind,
			Path:      namespace.Path,
		})
		namespace.reset()
	}

	return
}


type GitlabNamespace struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
	Path     string `json:"path"`
	ParentID int64  `json:"parent_id"`
	// If we have this field it means there is a valid tier
	MembersCountWithDescendants *int64 `json:"members_count_with_descendants"`
	AvatarURL                   string          `json:"avatar_url"`
	Kind                        string          `json:"kind"`
}
// Namespaces fetch namespaces
func NamespacesPage2(qc *QueryContext2,logger sdk.Logger, params url.Values) (np NextPage, namespaces []*GitlabNamespace, err error) {

	objectPath := "namespaces"

	np, err = qc.Get(logger, objectPath, params, &namespaces)
	if err != nil {
		return
	}

	return
}

func isValidTier(raw []byte) bool {
	return bytes.Contains(raw, []byte("members_count_with_descendants"))
}

func GroupUser(qc QueryContext, namespace *Namespace, userId string) (u *GitlabUser, err error) {

	sdk.LogDebug(qc.Logger, "group user access level", "namespace_name", namespace.Name, "namespace_id", namespace.ID, "user_id", userId)

	objectPath := sdk.JoinURL("groups", namespace.ID, "members", userId)

	_, err = qc.Get(objectPath, nil, &u)
	if err != nil {
		return
	}

	u.StrID = strconv.FormatInt(u.RefID, 10)

	return
}

// GroupProjects get group projects
func GroupProjectsIDs(qc QueryContext, group *Namespace) ([]string, error) {

	sdk.LogDebug(qc.Logger, "group projects", "group_name", group.Name, "group_id", group.ID)

	params := url.Values{}
	params.Set("with_projects", "true")

	objectPath := sdk.JoinURL("groups", url.QueryEscape(group.ID))

	var rr struct {
		Projects []*GitlabProject `json:"projects"`
	}

	_, err := qc.Get(objectPath, nil, &rr)
	if err != nil {
		return []string{}, err
	}

	projectIDs := make([]string, 0)
	for _, project := range rr.Projects {
		projectRefID := strconv.FormatInt(project.RefID, 10)
		projectID := sdk.NewWorkProjectID(qc.CustomerID, projectRefID, qc.RefType)
		projectIDs = append(projectIDs, projectID)
	}

	return projectIDs, nil
}
