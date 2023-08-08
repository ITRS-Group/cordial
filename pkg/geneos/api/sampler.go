package api

type Sampler struct {
	Entity
	Name string
}

// NewSampler returns a new Sampler. If there is an error connecting or
// the sampler is not configured in the Netprobe then an error is
// returned and s is set to nil.
func NewSampler(c *XMLRPCClient, entity, sampler string) (s *Sampler, err error) {
	s = &Sampler{Entity: Entity{XMLRPCClient: c, Name: entity}, Name: sampler}
	if exists, err := s.Exists(); err != nil || !exists {
		return nil, err
	}
	return
}

func (s *Sampler) Exists() (exists bool, err error) {
	if s == nil {
		return
	}
	return s.SamplerExists(s.Entity.Name, s.Name)
}

// XXX mount a plugin and start?

func (s *Sampler) WithPlugin() {

}
