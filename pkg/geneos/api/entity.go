package api

// Entity represents a Geneos Managed Entity
type Entity struct {
	*XMLRPCClient
	Name string
}

func (e *Entity) Exists() (exists bool, err error) {
	if e == nil {
		return
	}
	return e.ManagedEntityExists(e.Name)
}
