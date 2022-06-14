//go:build linux
// +build linux

package process

import (
	"github.com/itrs-group/cordial/pkg/samplers"
)

func (p ProcessSampler) initColumns() (cols samplers.Columns, columnnames []string, sortcol string, err error) {
	return p.ColumnInfo(nil)
}
