package icp

import (
	"context"
	"errors"
	"strings"
)

// utility function to make the API a bit simpler

// BaselineID returns the internal 24 hex-digit and the integer baseline
// ID for project where name matches the baseline view name. The name
// comparison is case insensitive. An error is returned if any
// underlying API call fails or if nothing is found.
func (i *ICP) BaselineID(ctx context.Context, project int, name string) (id string, baselineid int, err error) {
	b, _, err := i.BaselineViewsProject(ctx, project)
	if err != nil {
		return
	}
	for _, v := range b {
		if strings.EqualFold(v.Name, name) {
			return v.ID, v.BaselineID, nil
		}
	}
	err = errors.New("not found")
	return
}
