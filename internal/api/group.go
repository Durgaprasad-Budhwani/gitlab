package api

import (
	"net/url"
	"strconv"

	"github.com/pinpt/agent.next/sdk"
)

// Group internal group
type Group struct {
	ID       string
	FullPath string
}

// GroupsAll all groups
func GroupsAll(qc QueryContext) (allGroups []*Group, err error) {
	err = PaginateStartAt(qc.Logger, "", func(log sdk.Logger, paginationParams url.Values) (page PageInfo, _ error) {
		paginationParams.Set("per_page", "100")
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
func groups(qc QueryContext, params url.Values) (pi PageInfo, groups []*Group, err error) {

	sdk.LogDebug(qc.Logger, "groups request", "params", params)

	objectPath := "groups"

	var rgroups []struct {
		ID       int64  `json:"id"`
		FullPath string `json:"full_path"`
	}

	pi, err = qc.Request(objectPath, params, &groups)
	if err != nil {
		return
	}

	for _, group := range rgroups {
		groups = append(groups, &Group{
			ID:       strconv.FormatInt(group.ID, 10),
			FullPath: group.FullPath,
		})
	}

	return
}
