package api

import "strings"

type Dataview struct {
	Sampler
	Name  string
	Group string
}

// NewDataview returns a Dataview in d. If the connection to the API
// fails then a nil pointer is returned. If the dataview does not exist
// in the Netprobe it is also created.
func NewDataview(c *XMLRPCClient, entity, sampler, view string) (d *Dataview, err error) {
	group, name := "", view
	if strings.Contains(view, "-") {
		s := strings.SplitN(view, "-", 2)
		group, name = s[0], s[1]
	}
	d = &Dataview{
		Sampler: Sampler{
			Entity: Entity{
				XMLRPCClient: c,
				Name:         entity,
			},
			Name: sampler,
		},
		Name:  name,
		Group: group,
	}
	exists, err := d.Exists()
	if err != nil {
		return nil, err
	}
	if !exists {
		if err = d.CreateView(entity, sampler, name, group); err != nil {
			return nil, err
		}
	}
	return
}

func (d *Dataview) Exists() (exists bool, err error) {
	if d == nil {
		return
	}
	return d.ViewExists(d.Entity.Name, d.Sampler.Name, d.Name)

}

func (d *Dataview) Update() (err error) {
	return
}
