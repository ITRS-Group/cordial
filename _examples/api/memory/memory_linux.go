//go:build !windows

package memory

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"

	"github.com/itrs-group/cordial"
	extmemory "github.com/mackerelio/go-osstat/memory"
)

var log = cordial.Logger

func (p *MemorySampler) DoSample() error {
	log.Debug("called")

	table := [][]string{
		{"counter", "bytes"},
	}
	m, err := extmemory.Get()
	if err != nil {
		log.Error("error retrieving memory stats", slog.Any("error", err))
		os.Exit(1)
	}

	// use reflect to paint the structure into a table
	val := reflect.ValueOf(m).Elem()
	for i := 0; i < val.NumField(); i++ {
		n := val.Type().Field(i).Name
		v := val.Field(i).Interface()

		table = append(table, []string{n, fmt.Sprintf("%v", v)})
	}

	return p.UpdateTable(table[0], table[1:]...)
}
