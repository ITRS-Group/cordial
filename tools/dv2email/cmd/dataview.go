package cmd

import (
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/xpath"
)

func getDataview(d *xpath.XPath, gw *commands.Connection, em *config.Config) (dataview *commands.Dataview, err error) {
	dataview, err = gw.Snapshot(d, "", commands.Scope{Value: true, Severity: true})
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	// filter here

	headlines := match(dataview.Name, "headline-filter", "__headlines", em)
	if len(headlines) > 0 {
		nh := map[string]commands.DataItem{}
		for _, h := range headlines {
			h = strings.TrimSpace(h)
			for oh, headline := range dataview.Headlines {
				if ok, err := path.Match(h, oh); err == nil && ok {
					nh[oh] = headline
				}
			}
		}
		dataview.Headlines = nh
	}

	// the first column is either from `first-column` in config
	// (matched against the dataview name) or from the
	// environment variable _FIRSTCOLUMN or `rowname` and is
	// always the actual first column.
	var rowname string
	defaultRowName := match(dataview.Name, "first-column", "_firstcolumn", em)
	if len(defaultRowName) > 0 {
		rowname = defaultRowName[0]
	} else {
		rowname = em.GetString("_firstcolumn", config.Default("rowname"))
	}
	// set the default, may be overridden below but then reset
	// to the same value
	dataview.Columns[0] = rowname

	cols := match(dataview.Name, "column-filter", "__columns", em)
	if len(cols) > 0 {
		nc := []string{rowname}
		for _, c := range cols {
			c = strings.TrimSpace(c)
			for _, oc := range dataview.Columns {
				if oc == rowname {
					continue
				}
				if ok, err := path.Match(c, oc); err == nil && ok {
					nc = append(nc, oc)
				}
			}
		}

		dataview.Columns = nc
	}

	rows := match(dataview.Name, "row-filter", "__rows", em)
	if len(rows) > 0 {
		nr := map[string]map[string]commands.DataItem{}
		for _, r := range rows {
			r = strings.TrimSpace(r)
			for rowname, row := range dataview.Table {
				if ok, err := path.Match(r, rowname); err == nil && ok {
					nr[rowname] = row
				}
			}
		}
		dataview.Table = nr
	}

	// default sort rows
	sort.Strings(dataview.Rows)

	asc := true
	matches := matchForName(dataview.Name, "row-order")
	if len(matches) > 0 {
		m := matches[0]
		switch {
		case strings.HasSuffix(m, "-"):
			asc = false
			m = m[:len(m)-1]
		case strings.HasSuffix(m, "+"):
			m = m[:len(m)-1]
			fallthrough
		default:
			asc = true
		}

		// if the row-order is for a column that is used as the
		// rowname (decided above in Column ordering) then sort
		// the data.Rows slice directly based on value and
		// not a cell in the row
		if m == "rowname" || m == dataview.Columns[0] {
			sort.Slice(dataview.Rows, func(i, j int) bool {
				if asc {
					return dataview.Rows[i] < dataview.Rows[j]
				} else {
					return dataview.Rows[j] < dataview.Rows[i]
				}
			})
		} else {
			sort.Slice(dataview.Rows, func(i, j int) bool {
				r := dataview.Rows
				a := dataview.Table[r[i]][m].Value
				af, _ := strconv.ParseFloat(a, 64)
				b := dataview.Table[r[j]][m].Value
				bf, _ := strconv.ParseFloat(b, 64)
				if a == b {
					if asc {
						return a < b
					} else {
						return a > b
					}
				}
				if asc {
					return af < bf
				}
				return bf < af
			})
		}
	}

	return
}
