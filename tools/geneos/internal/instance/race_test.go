package instance_test

import (
	"sync"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/san"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

// TestConcurrentInstanceUse is for the race detector. Commands that work
// on several instances at once share the cached instance values between
// goroutines, which load, unload, write and list them at the same time.
func TestConcurrentInstanceUse(t *testing.T) {
	root := newTestRoot(t)
	_ = root
	cts := []*geneos.Component{&licd.Component, &fa2.Component, &san.Component}
	names := []string{"one", "two", "three"}
	for _, ct := range cts {
		for _, name := range names {
			i, err := instance.Add(geneos.LOCAL, ct, name, instance.Insecure(true))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = i.Unload() })
		}
	}

	var wg sync.WaitGroup
	for g := range 12 {
		wg.Go(func() {
			for n := range 20 {
				ct := cts[(g+n)%len(cts)]
				name := names[(g+n/2)%len(names)]
				switch g % 6 {
				case 0:
					if i, err := instance.Get(ct, name); err == nil {
						_ = i.Loaded()
					}
				case 1:
					if i, err := instance.Get(ct, name); err == nil {
						if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
							t.Error(resp.Err)
						}
					}
				case 2:
					for _, i := range instance.Instances(geneos.LOCAL, nil) {
						_ = i.Loaded()
						_ = i.Home()
					}
				case 3:
					if i, err := instance.Get(ct, name); err == nil {
						_ = i.Unload()
					}
				case 4:
					_ = instance.GetAllPorts(geneos.LOCAL)
				case 5:
					i, err := instance.Get(ct, name)
					if err != nil {
						continue
					}
					cf := i.Config()
					config.Set(cf, "autostart", n%2 == 0)
					i.SetConfig(cf)
					resps := instance.DoInstances("test", []geneos.Instance{i}, func(i geneos.Instance, _ ...any) *responses.General {
						return responses.New[responses.General](i)
					})
					_ = resps
				}
			}
		})
	}
	wg.Wait()
}
