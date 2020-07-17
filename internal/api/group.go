package api

import (
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/sdk"
)

// Group internal group
type Group struct {
	ID       string
	Name     string
	FullPath string
}

// GroupsAll all groups
func GroupsAll(qc QueryContext) (allGroups []*Group, err error) {
	err = Paginate(qc.Logger, "", time.Time{}, func(log sdk.Logger, paginationParams url.Values, t time.Time) (np NextPage, _ error) {
		paginationParams.Set("top_level_only", "true")

		pi, groups, err := groups(qc, paginationParams)
		if err != nil {
			return pi, err
		}
		allGroups = append(allGroups, groups...)
		return pi, nil
	})
	return
}

// Groups fetch groups
func groups(qc QueryContext, params url.Values) (np NextPage, groups []*Group, err error) {

	sdk.LogDebug(qc.Logger, "groups request", "params", params)

	objectPath := "groups"

	var rgroups []struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		FullPath string `json:"full_path"`
	}

	np, err = qc.Get(objectPath, params, &rgroups)
	if err != nil {
		return
	}

	for _, group := range rgroups {
		groups = append(groups, &Group{
			ID:       strconv.FormatInt(group.ID, 10),
			Name:     group.Name,
			FullPath: group.FullPath,
		})
	}

	return
}
