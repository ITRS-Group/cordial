package instance

import (
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type SortInstanceResults struct {
	Instances []geneos.Instance
	Results   []interface{}
}

func (s SortInstanceResults) Len() int { return len(s.Instances) }

func (s SortInstanceResults) Swap(i, j int) {
	s.Instances[i], s.Instances[j] = s.Instances[j], s.Instances[i]
	s.Results[i], s.Results[j] = s.Results[j], s.Results[i]
}

func (s SortInstanceResults) Less(i, j int) bool {
	ci := s.Instances[i]
	cj := s.Instances[j]

	switch {
	case ci.Host().String() != cj.Host().String():
		return ci.Host().String() < cj.Host().String()
	case ci.Type().String() != cj.Type().String():
		return ci.Type().String() < cj.Type().String()
	case ci.Name() != cj.Name():
		return ci.Name() < cj.Name()
	default:
		return false
	}
}
