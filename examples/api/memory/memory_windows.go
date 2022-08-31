// +build windows
package memory

import (
	"fmt"
	"log"
	"reflect"

	_ "github.com/StackExchange/wmi"
	extmemory "github.com/mackerelio/go-osstat/memory"
)

func (p *MemorySampler) DoSample() error {
	table := [][]string{
		{"counter", "bytes"},
	}
	m, err := extmemory.Get()
	if err != nil {
		log.Fatal(err)
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
