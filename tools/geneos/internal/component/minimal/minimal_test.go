/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package minimal

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// TestDefaultHomeIsRealHome checks that the registered default home is
// the directory that is really used for a new instance, which has
// always been `netprobe/minimals/NAME`.
func TestDefaultHomeIsRealHome(t *testing.T) {
	root := geneostest.NewRoot(t, "", name)

	// the factory sets the home of an instance to the directory in use,
	// so render the registered default to check it
	var defaultHome string
	for _, d := range Component.Defaults {
		if v, ok := strings.CutPrefix(d, "home="); ok {
			tmpl, err := template.New("home").Funcs(template.FuncMap{"join": geneos.LOCAL.Join}).Parse(v)
			if err != nil {
				t.Fatal(err)
			}
			var b strings.Builder
			if err = tmpl.Execute(&b, map[string]string{"root": root, "name": "mini1"}); err != nil {
				t.Fatal(err)
			}
			defaultHome = b.String()
		}
	}

	i, err := instance.Add(geneos.LOCAL, &Component, "mini1", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(root, "netprobe", "minimals", "mini1")
	if defaultHome != want {
		t.Errorf("default home = %q, want %q", defaultHome, want)
	}
	if got := i.Home(); got != want {
		t.Errorf("home of a new instance = %q, want %q", got, want)
	}
	if got := config.Get[string](i.Config(), "home"); got != want {
		t.Errorf("saved home = %q, want %q", got, want)
	}
	if st, err := os.Stat(want); err != nil || !st.IsDir() {
		t.Errorf("instance directory %q does not exist: %v", want, err)
	}
	if st, err := os.Stat(filepath.Join(root, "netprobe", "minimals")); err != nil || !st.IsDir() {
		t.Errorf("instances directory is not netprobe/minimals: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "netprobe", "netprobes")); err == nil {
		t.Error("a new minimal installation created netprobe/netprobes")
	}
}

// TestExistingLocationsKept checks that instances in the older locations
// are still found where they are.
func TestExistingLocationsKept(t *testing.T) {
	root := geneostest.NewRoot(t, "", name)
	for _, dir := range []string{
		filepath.Join(root, "minimal", "minimals", "legacy"),
		filepath.Join(root, "netprobe", "netprobes", "oldhome"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	i, err := instance.GetWithHost(geneos.LOCAL, &Component, "legacy")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	if got, want := i.Home(), filepath.Join(root, "minimal", "minimals", "legacy"); got != want {
		t.Errorf("home = %q, want existing %q", got, want)
	}

	// an instance that was configured with the old default is also kept
	i, err = instance.GetWithHost(geneos.LOCAL, &Component, "oldhome")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	config.Set(i.Config(), "home", filepath.Join(root, "netprobe", "netprobes", "oldhome"))
	if got, want := i.Home(), filepath.Join(root, "netprobe", "netprobes", "oldhome"); got != want {
		t.Errorf("home = %q, want configured %q", got, want)
	}
}
