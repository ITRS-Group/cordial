//go:build linux

package process

import (
	"github.com/itrs-group/cordial/pkg/geneos/samplers"
)

func (p ProcessSampler) initColumns() (cols samplers.Columns, columnnames []string, sortcol string, err error) {
	return p.ColumnInfo(nil)
}
