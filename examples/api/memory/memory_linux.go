//go:build linux
// +build linux

package memory

import (
	"fmt"
	"reflect"

	extmemory "github.com/mackerelio/go-osstat/memory"
	"github.com/rs/zerolog/log"
)

func (p *MemorySampler) DoSample() error {
	log.Debug().Msg("called")

	table := [][]string{
		{"counter", "bytes"},
	}
	m, err := extmemory.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("")
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
